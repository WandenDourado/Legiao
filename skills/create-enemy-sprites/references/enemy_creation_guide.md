# Guia de criação de inimigos — Legiao

Complementa `SKILL.md` com a direção de arte e a estrutura dos prompts. O
contrato visual do jogo inteiro está em `doc/art_style.md` e prevalece sobre
qualquer coisa aqui.

## 1. Escolha do modo

| | Radial | Direcional |
|---|---|---|
| Sheets | 1 vista | 3 linhas (S, W, N) |
| Direção | rotação em runtime, 360° livre | snap em 4 direções, espelha W→E |
| Frame | quadrado, pivô no centro | `128x192`, âncora no pé |
| Serve para | amorfos, radialmente simétricos, quadrúpedes | bípedes, tudo com rosto de frente |
| Não serve para | rosto frontal, superfície rígida | nada — só é mais caro |

O modo radial tem duas fraquezas estruturais, e vale dizê-las ao usuário antes
de escolher:

**A luz gira junto com o sprite.** Numa geleia isso passa; numa armadura ou num
crânio, denuncia. A mitigação está na seção 3.

**O rosto some.** Visto de cima você olha as costas. Tentativas de trazer o
rosto de volta empurram o gerador para uma câmera frontal, que é incompatível
com rotação. Se o medo do inimigo mora na cara, use o modo direcional.

## 2. Escala

A régua é a mesma do resto do jogo: **porta de 150px, herói de 186px de altura**.
Nunca "o tamanho que pareceu bom".

| Inimigo | Corpo em tela | Frame | RenderScale | Raio |
|---|---|---|---|---|
| Slime | ~100px de largura | 256 | 0.575 | 45 |
| Lobo | ~180px de comprimento | 256 | 0.9 | 50 |

O `RenderScale` fica **abaixo de 1** de propósito: o frame é maior que o
tamanho de tela para que o runtime reduza em vez de ampliar. Ampliar sprite
pintado dá bloco — foi o que aconteceu na primeira versão, com frames de 128px
e o lobo desenhado a 1.8x.

O `RenderScale` é a ferramenta de legibilidade. Se o bicho não lê no jogo, ele
está pequeno demais — **não** estreito demais. Alargar a anatomia para encher o
frame é a tentação que produziu a pose em X do lobo v2.

Um corpo alongado ocupa pouca área do frame quadrado por definição: o lobo usa
12% contra 33% do slime. Isso não é defeito, é consequência de ser comprido.

## 3. Luz no modo radial

Regra: **o padrão de valor vem das marcas do bicho, não da direção da luz.**

O `doc/art_style.md` fixa luz superior-esquerda para todo asset do jogo. O modo
radial é a exceção, por necessidade física: um sprite que gira leva a luz junto.

Pedir "iluminação zenital" não funciona — foi tentado duas vezes e o gerador
pintou luz superior-esquerda nas duas. O que funciona é pedir **luz chapada, de
céu encoberto, sem sombra**, e mandar todo o contraste vir da pelagem, da
casca, do padrão do corpo. Sem luz direcional, a rotação não tem o que revelar.

No slime sobrou um brilho fora de centro e passou, porque geleia é amorfa. Num
corpo com anatomia definida isso não passaria.

## 4. Paleta

Ver a seção "Colour language" do `SKILL.md`. Em resumo: herói é o mais claro da
tela, inimigo separa do terreno por matiz **ou** por valor — declare qual — e
separar por valor tem o risco de perder o contraste interno.

Famílias já em uso:

- **Slime**, verde-jade em torno de `#059A4F`, matiz 150 contra os 85 da grama.
  Luminância 111 contra 122 da grama — praticamente igual —, então quem carrega
  a separação é o matiz somado à saturação de 0.94. Foi obtido rotacionando o
  matiz da versão carmesim original, sem gerar nada de novo.
- **Lobo**, carvão `#23282F → #3D444C → #99A3AC`, separação por valor, com olhos
  `#C4241A` marcando a família inimiga.

## 5. Como assustar

Sem sangue, sem ferida, sem víscera. O medo vem da silhueta e da postura, que é
o que sobrevive à escala de jogo:

- **Pelo eriçado na espinha.** De cima, a espinha é o que se vê — é o melhor
  investimento de leitura que existe no modo radial.
- Orelhas coladas para trás, focinho franzido, dentes à mostra.
- Corpo magro e anguloso; volume fofo é o inimigo do medo.
- Garras abertas, patas grandes.
- Olhos estreitos e vermelhos com um ponto de luz duro — **nunca brilho difuso**,
  que o `art_style.md` proíbe e que o runtime é quem deve fazer.

## 6. Estrutura do arquivo de prompt

Cada prompt vira um arquivo em `work/enemy-sprites/prompt-N-<nome>.md` com:

1. **O que a versão anterior errou**, com os números medidos numa tabela. É o
   que o gerador precisa ler para convergir, e o que uma auditoria futura
   precisa para entender por que a arte é como é.
2. **As correções**, explicando a causa de cada defeito.
3. **O prompt em si**, num bloco de citação contínuo, pronto para colar.
4. **O checklist do resultado**, dizendo quais itens são medidos por script e
   quais exigem olhar humano, e o que é regeração versus reparo.

Manter os prompts reprovados. A sequência v1→v4 do lobo é o registro de por que
cada regra existe; apagar as tentativas apaga o motivo.

## 7. Ciclos de animação

| Tipo de corpo | Frames | Tempo/frame | Estrutura |
|---|---|---|---|
| Amorfo (pulso) | 6 | 0.11 | um ciclo de squash-and-stretch |
| Quadrúpede (galope) | 8 | 0.07 | **duas** meias-passadas alternando a guia |

O ciclo roda **no lugar**: o deslocamento vem do código. Um bicho que anda
dentro da própria célula descentra o pivô e faz o sprite orbitar ao girar.

Movimento entre frames medido: slime 10,5%, lobo 19,6%. Abaixo de ~6% a
animação lê como parada — `check_gait.py` avisa.

## 8. Casos de teste

Os quatro lobos e os dois slimes gerados nesta produção estão em
`work/enemy-sprites/` e servem de suite de regressão para os scripts:

| Versão | Defeito | Quem pega |
|---|---|---|
| slime v1 | oscilação de squash de 1,8x, estouro do círculo inscrito | `validate_radial` |
| lobo v1 | 89% do corpo em preto chapado, câmera frontal | `validate_radial` (valor) + olho |
| lobo v2 | pose em X, cintura com razão 0,59 | `check_gait` |
| lobo v3 | guia fixa, desequilíbrio 29 | `check_gait` |
| lobo v4 | — aprovado | — |

Ao mexer nos limiares dos scripts, rode contra esses casos: o v4 tem que passar
e os anteriores têm que reprovar pelo motivo certo.
