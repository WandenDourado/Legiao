#version 330
// Mesma borda desbotada do terrain_blend, mas com UMA troca de shader por
// MATERIAL em vez de uma por CELULA.
//
// O terrain_blend recebe a mascara de vizinhos em uniform, e uniform so muda
// entre draw calls: por isso cada celula era embrulhada num Begin/EndShaderMode
// proprio, que esvazia o batch do rlgl. No world_03 eram 1.104 draw calls por
// quadro so de chao.
//
// Aqui a mascara vem de uma TEXTURA do tamanho da grade (um texel por celula) e
// o indice da celula chega pela COR do vertice, que o DrawTexturePro deixa
// variar por quad sem quebrar o batch. Todas as celulas de um material saem num
// bloco so.
in vec2 fragTexCoord;
in vec4 fragColor;
out vec4 finalColor;

uniform sampler2D texture0;   // a textura do material
uniform sampler2D maskTex;    // W x H, um texel por celula: R = bordas, G = cantos
uniform vec4 colDiffuse;

uniform vec2 gridSize;        // largura e altura do mapa, em celulas
uniform float edgeWidth;      // a rampa desta PILHA (nao global)
uniform float spanF;          // quantas celulas a folha cobre num lado

// bitAt devolve 1.0 quando o bit indicado esta ligado no byte empacotado.
// O byte chega como 0..1 vindo da textura, entao volta a 0..255 antes.
float bitAt(float packed, float bit) {
    float v = floor(packed * 255.0 + 0.5);
    return floor(mod(v / exp2(bit), 2.0));
}

// edgeWidth pode ser 0.01 (a borda dura de proposito da pilha do castelo).
// smoothstep(a,b,x) e INDEFINIDO com a >= b; no mobile isso vira NaN e apaga a
// celula inteira. Os limites sao derivados de edgeWidth para que uma pilha de
// borda dura continue dura, e nao para que fique macia.
float rampLo(float w) { return min(0.01, w * 0.5); }
float rampHi(float w) { return max(w, rampLo(w) + 0.002); }
float cornerLo(float w) { return min(0.02, w * 0.5); }
float cornerHi(float w) { return max(w, cornerLo(w) + 0.002); }
float side(float uv, float linked) {
    return mix(smoothstep(rampLo(edgeWidth), rampHi(edgeWidth), uv), 1.0, linked);
}

void main() {
    // A COR do vertice carrega a celula: r e g sao x e y em 0..255. Nao ha
    // outro canal por quad no DrawTexturePro, e um uniform aqui recriaria
    // exatamente o problema que este shader existe para resolver.
    vec2 cell = floor(fragColor.rg * 255.0 + 0.5);

    // uv local DENTRO da celula, 0..1. Derivado do span em vez de fract():
    // fract devolveria 0 na aresta final da celula em vez de 1, e isso
    // imprimiria uma linha de um pixel em cada emenda.
    vec2 window = vec2(mod(cell.x, spanF), mod(cell.y, spanF));
    vec2 uv = clamp(fragTexCoord * spanF - window, 0.0, 1.0);

    vec4 packed = texture(maskTex, (cell + 0.5) / gridSize);

    // ATALHO DA CELULA CERCADA. Os oito vizinhos ligados sao os quatro bits de
    // cada canal em 1, ou seja 15. Nesse caso nao existe borda para desbotar:
    // `side()` daria 1 nos quatro lados, nenhum termo de canto entraria (todos
    // exigem um canto DESLIGADO) e o dither zeraria junto com `ramp`. A conta
    // inteira abaixo terminaria em mask = 1,0.
    //
    // Sair aqui pula oito smoothstep, quatro length() e o sin() do ruido, e
    // este e o caso da GRANDE MAIORIA dos fragmentos: ~90% das celulas dos
    // mapas de campanha estao no meio do proprio material. Depois da oclusao
    // do lado do Go (TerrainRenderer.hidden), que apaga as camadas ESCONDIDAS,
    // este atalho e o que barateia as que sobraram — as visiveis.
    if (floor(packed.r * 255.0 + 0.5) > 14.5 && floor(packed.g * 255.0 + 0.5) > 14.5) {
        finalColor = texture(texture0, fragTexCoord) * colDiffuse;
        return;
    }

    // Mesma ordem de terrain_mask.go: bordas N, L, S, O; cantos NO, NE, SE, SO.
    vec4 edge = vec4(bitAt(packed.r, 0.0), bitAt(packed.r, 1.0),
                     bitAt(packed.r, 2.0), bitAt(packed.r, 3.0));
    vec4 corner = vec4(bitAt(packed.g, 0.0), bitAt(packed.g, 1.0),
                       bitAt(packed.g, 2.0), bitAt(packed.g, 3.0));

    float mask = min(min(side(uv.y, edge.x), side(1.0-uv.x, edge.y)),
                     min(side(1.0-uv.y, edge.z), side(uv.x, edge.w)));
    if (edge.x > 0.5 && edge.w > 0.5 && corner.x < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv)));
    if (edge.x > 0.5 && edge.y > 0.5 && corner.y < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(1.0,0.0))));
    if (edge.z > 0.5 && edge.y > 0.5 && corner.z < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(1.0,1.0))));
    if (edge.z > 0.5 && edge.w > 0.5 && corner.w < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(0.0,1.0))));

    // O dither de sempre, ancorado no texel para nao ferver com a camera, e
    // pesado pela rampa para nao cobrir o interior da celula.
    vec2 texel = floor(uv * 128.0);
    float n = fract(sin(dot(texel, vec2(12.9898,78.233)))*43758.5453);
    float ramp = mask * (1.0 - mask) * 4.0;
    mask = clamp(mask + (n-0.5)*0.16*ramp, 0.0, 1.0);

    vec4 src = texture(texture0, fragTexCoord);
    // fragColor NAO multiplica a cor: ele carrega o indice da celula, nao um
    // tom. Multiplicar pintaria o chao com as coordenadas dele.
    finalColor = vec4(src.rgb, src.a*mask)*colDiffuse;
}
