"""Helpers compartilhados pelos scripts de mapa.

Convencao de coordenadas: tudo que e argumento de script esta em CELULAS
(coluna, linha). Objetos no JSON do Tiled ficam em pixels, convertidos aqui.
"""
import json

# Os materiais de chao, por PILHA (ver internal/tilemap/terrain_mask.go).
# Material de pilhas diferentes nunca se pinta um sobre o outro, entao a pilha
# a que um material pertence decide o que ele vai dissolver na borda:
#
#   chao verde   1 grama      -> 2 terra      -> 3 pedra
#   bioma escuro 6 terra nua  -> 5 grama rala -> 4 grama escura
#
# A grama (1) e desenhada como base sob TUDO, inclusive sob o bioma escuro.
# E por isso que uma celula escura encostada numa clara ja le, sozinha, como
# grama clara -> terra nua -> grama rala -> grama escura: a transicao de bioma
# nao se pinta, se declara.
TERRAIN = {
    "grass": 1, "dirt": 2, "stone": 3,
    "dark_grass": 4, "sparse_grass": 5, "bare_soil": 6,
    # A grama VIVA da mata: primeiro estagio da sequencia do bioma escuro, e nao
    # uma segunda grama de vila. Ela mora na pilha escura de proposito - e o que
    # deixa o motor rampar a transicao sozinho.
    "forest_grass": 7,
    # A estrada da mata: mesma arte da terra da vila, mas na pilha escura, para
    # a borda dela desbotar contra a grama viva e nao contra a grama do mapa 1.
    "forest_path": 8,
    # O chao CONSTRUIDO da fortaleza (mapa 3), no topo da mesma pilha escura:
    # cascalho de cerco e laje negra. Eles ficam nessa pilha, e nao numa
    # propria, porque material de pilhas diferentes nunca se pinta um sobre o
    # outro - a esplanada terminaria num degrau duro contra a terra morta.
    "siege_gravel": 9, "dark_flagstone": 10,
}
TERRAIN_NAME = {v: k for k, v in TERRAIN.items()}

# A qual pilha cada material pertence, para quem precisa saber se dois
# materiais conversam na borda.
TERRAIN_STACKS = (("grass", "dirt", "stone"),
                  ("forest_path", "forest_grass", "dark_grass", "sparse_grass",
                   "bare_soil", "siege_gravel", "dark_flagstone"))
TERRAIN_STACK_OF = {name: i for i, stack in enumerate(TERRAIN_STACKS) for name in stack}

# Chao CONSTRUIDO: terra pisada e pedra assentada sao marcas que alguem deixou
# no chao verde, e nada nasce em cima delas. Todo o resto e chao natural, e
# isso inclui o bioma escuro inteiro — terra nua (6) e o fundo de uma mata, nao
# uma estrada. Confundir os dois reprova toda arvore de um mapa escuro.
PAVED = {TERRAIN["dirt"], TERRAIN["stone"], TERRAIN["forest_path"],
         # Cascalho e laje sao obra, nao chao de mata: nada brota neles. Sao os
         # dois unicos materiais da pilha escura que entram aqui.
         TERRAIN["siege_gravel"], TERRAIN["dark_flagstone"]}

# `zones` guarda RETANGULOS nomeados, e nao pontos: territorio de guarnicao,
# vao de barricada e a area da fortaleza. Um mapa de defesa de territorio
# precisa dizer ONDE cada coisa vale, e ponto solto nao diz.
OBJECT_LAYERS = ("vegetation", "buildings", "fences", "props", "spawn", "trails",
                 "portals", "zones")


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def save(tmap, path):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(tmap, f, indent=1)


def tile_layer(tmap, name):
    for layer in tmap["layers"]:
        if layer.get("name") == name and layer.get("type") == "tilelayer":
            return layer
    raise SystemExit(f"camada de tiles '{name}' nao existe no mapa")


def object_layer(tmap, name, create=False):
    for layer in tmap["layers"]:
        if layer.get("name") == name and layer.get("type") == "objectgroup":
            return layer
    if not create:
        return None
    layer = {"id": tmap.get("nextlayerid", 100), "name": name, "opacity": 1,
             "type": "objectgroup", "visible": True, "x": 0, "y": 0, "objects": []}
    tmap["nextlayerid"] = tmap.get("nextlayerid", 100) + 1
    tmap["layers"].append(layer)
    return layer


def add_object(tmap, layer_name, name, cell_x, cell_y, kind="", pixel=None):
    """Coloca um objeto nomeado. Por padrao no centro-base da celula."""
    layer = object_layer(tmap, layer_name, create=True)
    tw, th = tmap["tilewidth"], tmap["tileheight"]
    x, y = pixel if pixel else (int(cell_x * tw + tw / 2), int((cell_y + 1) * th))
    oid = tmap.get("nextobjectid", 1)
    layer["objects"].append({"id": oid, "name": name, "type": kind, "x": x, "y": y,
                             "width": 0, "height": 0, "visible": True})
    tmap["nextobjectid"] = oid + 1
    return x, y


def cells_of(tmap, layer_name, predicate=lambda gid: gid != 0):
    layer = tile_layer(tmap, layer_name)
    w = tmap["width"]
    return {(i % w, i // w) for i, gid in enumerate(layer["data"]) if predicate(gid)}


def set_cell(layer, width, col, row, value, height=None):
    if col < 0 or row < 0 or col >= width:
        return False
    i = row * width + col
    if i >= len(layer["data"]):
        return False
    layer["data"][i] = value
    return True


def get_cell(layer, width, col, row, default=0):
    if col < 0 or row < 0 or col >= width:
        return default
    i = row * width + col
    return layer["data"][i] if i < len(layer["data"]) else default


def manifests_for(tmap, manifest_paths):
    """Carrega manifestos e devolve name -> piece."""
    pieces = {}
    for path in manifest_paths:
        data = load(path)
        for name, piece in data.get("pieces", {}).items():
            pieces[name] = piece
    return pieces


def blocked_cells(tmap, manifest_paths=()):
    """Celulas solidas: camada collision + footprints dos manifestos.

    Aproximacao grosseira para auditoria de layout: o motor colide contra os
    RETANGULOS do manifesto, nao contra celulas. Aqui uma celula conta como
    bloqueada assim que um footprint a toca, o que superestima de proposito --
    e a direcao segura para checar se um caminho existe.
    """
    tw, th = tmap["tilewidth"], tmap["tileheight"]
    blocked = set(cells_of(tmap, "collision"))
    pieces = manifests_for(tmap, manifest_paths)
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup":
            continue
        for obj in layer.get("objects", []):
            piece = pieces.get(obj.get("name"))
            if not piece or not piece.get("collision"):
                continue
            for fp in footprints_of(piece):
                x, y = obj["x"] + fp["offsetX"], obj["y"] + fp["offsetY"]
                w, h = fp["width"], fp["height"]
                for row in range(int(y // th), int((y + h - 0.01) // th) + 1):
                    for col in range(int(x // tw), int((x + w - 0.01) // tw) + 1):
                        ow = min(x + w, (col + 1) * tw) - max(x, col * tw)
                        oh = min(y + h, (row + 1) * th) - max(y, row * th)
                        if ow > 0 and oh > 0:
                            blocked.add((col, row))
    return blocked


def footprints_of(piece):
    """Todo retangulo de colisao de uma peca, nas duas formas do manifesto."""
    if not piece.get("collision"):
        return []
    many = piece.get("collisionFootprints")
    if many:
        return many
    single = piece.get("collisionFootprint")
    return [single] if single else []


def spawn_cell(tmap):
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup":
            continue
        for obj in layer.get("objects", []):
            if obj.get("name") == "player_spawn":
                return int(obj["x"]) // tmap["tilewidth"], int(obj["y"]) // tmap["tileheight"]
    return None
