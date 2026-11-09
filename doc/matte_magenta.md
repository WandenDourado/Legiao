# Matte magenta: como tirar o fundo sem deixar rosa

Toda arte gerada neste projeto volta sobre `#FF00FF` chapado (o gerador não
entrega alfa, e cinza não recorta — manto creme e capa branca caem na mesma
faixa). Isso vale para sprites de personagem, inimigos, cenário e retratos.
Este documento junta o que custou erro para aprender ao tirar esse fundo,
especialmente na limpeza dos retratos de 28/09/2026. Cada pipeline tem o seu
script de keying, mas as regras abaixo valem para todos.

Skill operacional: `skills/remove-magenta-matte/SKILL.md`.

## 1. O rosa que sobra não é magenta puro

Tirar os pixels "que são magenta" só pega a ponta do problema. O que fica
visível depois é o matte **misturado** com a pintura, em toda proporção:

| Onde | Por quê | Exemplo medido |
|---|---|---|
| Anel de 1–3 px no contorno | antialias do gerador + a redução LANCZOS espalha a mistura | `(66, 6, 66)`: matte + traço preto |
| Pontas de cabelo, bordas de capa | o gerador **pinta** luz de rebote magenta na arte | `(247, 107, 164)` num cacho castanho |
| Frestas **dentro** da figura | bolsão de fundo pequeno demais para virar transparente; ficou opaco e misturado | `(142, 75, 113)` entre mechas loiras, 20 px da borda |

A terceira linha é a que escapa de toda limpeza que só olha a borda. Foi
achada pelo Gui no jogo, no cabelo da Sacerdotisa, depois de a borda já estar
limpa.

## 2. O discriminante é o VERDE

O matte real medido nas gerações é `(236, 3, 226)` — não `(255, 0, 255)`.
Usar o valor medido na conta de desmistura evita um tom azulado residual.

"R e B acima de G" descreve magenta **e violeta**. Essa regra comeu 26.366 px da
chama do Necromante. Separa-se pelo verde e pelo azul dominante:

| | R | G | B |
|---|---|---|---|
| Matte | 236 | **3** | 226 |
| Chama, núcleo claro | 229 | **120** | 249 |
| Chama, faixa escura | 106 | **31** | 183 |

Violeta legítimo tem `B − R > 0,30·B + 10` e/ou G bem acima de zero.

## 3. Não apagar: trocar a cor, manter a luminância

Apagar ou zerar o pixel rosa come o traço do personagem, que é justamente a
borda. O que funciona:

1. Estimar a fração de matte: `k = min((R−G)/(236−3), (B−G)/(226−3))`.
2. Desmisturar para achar a luminância que a pintura tinha:
   `C = (obs − k·M) / (1 − k)`.
3. Pintar o pixel com a **cor do miolo vizinho** (média dos pixels opacos e
   limpos logo para dentro), escalada para a luminância de `C`.
4. Só no anel mais externo (≤ 2 px) o alfa cai para `A·(1 − k)` — ali o rosa
   era cobertura de fundo, não tinta.

Resultado: traço escuro continua escuro, brilho continua brilho, e o cacho
castanho com luz rosa vira cacho castanho com luz quente.

Não usar o *despill* clássico que só limita R e B a `G + 60`: ele deixa o rosa
escuro intacto sobre cabelo escuro (foi o que sobrou na Paladina).

## 4. Proteger a pintura que é violeta de propósito

Critério de **contexto**, não de cor isolada: olha-se o miolo da figura numa
janela de ~61 px. Se mais de 12% dele já é violeta/magenta pintado, o rosa da
borda ali é luz pintada e fica. Foi o que salvou a chama, a mão e a luz no
rosto do Necromante.

Para o passo de miolo, um personagem com mais de 1% do miolo violeta entra em
**modo estrito**: só conta matte escuro (G < 25). No Necromante o rosado do
peito e do pescoço é luz da chama; sem o modo estrito o passo de miolo marcava
centenas de focos em cima dela.

## 5. Achar o rosa de dentro: anomalia local

Frestas não estão perto da borda, então nenhuma faixa de borda chega nelas.
O critério que funciona é: o pixel tem **bem mais** excesso de R e B sobre G
(`s = min(R−G, B−G)`) do que a mediana dos vizinhos opacos numa janela de
15 px (`s − mediana > 10`). Luz pintada é larga e acompanha a vizinhança;
matte preso é um ponto que destoa dela.

## 6. Como conferir (o que já enganou)

- **Conferir sobre verde, não só sobre cinza escuro.** O rosa das frestas da
  Sacerdotisa passava despercebido no fundo escuro e salta sobre verde (que
  também é o que o jogo mostra por trás, no gramado).
- **Zoom de 3–4x, com o antes e depois lado a lado.** Na miniatura não se vê
  nada.
- **Mapa de diferença** (pixels alterados pintados de verde sobre a arte
  escurecida) no personagem de risco. Foi ele que mostrou que a primeira versão
  do passo de miolo pintava a chama do Necromante inteira.
- **Número fora da curva no lote é o sinal.** Rode o elenco inteiro e compare;
  medido sozinho, um número não diz nada.
- `audit_matte.py` imprime borda/miolo/violeta por arquivo e grava um overlay
  com os focos circulados.

## 7. Regras de operação

- **Sempre rodar a partir do original**, nunca em cima de uma saída já limpa.
  Originais dos retratos: `work/portraits/antes_despill_2026-09-28/`.
- **Backup nunca dentro de `assets/`**: o build Android faz `xcopy /E` da
  pasta inteira e arquivo morto entra no APK.
- Se o Windows recusar a escrita (`OSError: Invalid argument`), o PNG está
  aberto em algum programa: gravar num temporário e copiar depois de fechar.
- Ao mandar um script do container para a máquina do Gui, conferir o md5 no
  destino: uma vez o `device_commit_files` regravou uma versão antiga do mesmo
  arquivo.

## 8. Qual ferramenta em qual pipeline

| Pipeline | Keying | Rosa residual |
|---|---|---|
| Retratos (diálogo e seleção) | `work/portraits/key_and_frame.py` | automático: chama `despill_matte.py` antes de salvar |
| Sprites de personagem 128x192 | `slice_and_stitch.py` | `auto_repair.py` (a escala dos quadros é pequena demais para as janelas do `despill_matte.py`) |
| Inimigos | scripts da `create-enemy-sprites` | idem sprites |
| Cenário (atlas) | `create-tiled-assets/scripts/key_matte.py` | desmistura própria + edge bleed; ver a tabela de defeitos da skill |
| Qualquer PNG já recortado | — | `despill_matte.py`, depois `audit_matte.py` |

Os parâmetros de `despill_matte.py` (faixa 8 px, janelas 15 e 61 px) foram
afinados em telas de 1024–1152 px de largura. Para arte muito menor, escalar as
janelas na mesma proporção antes de confiar no resultado.
