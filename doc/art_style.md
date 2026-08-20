# Guia de Estilo Visual — Legiao

Este documento descreve o estilo visual estabelecido do jogo, medido a partir
dos assets já aprovados (`assets/sprites/`, `assets/tilesets/`). Serve como
contrato de estilo para gerar novos assets — personagens, cenário ou props.
Para regras de montagem e integração técnica, ver `tileset_spec.md` e
`tilemap.md`; este documento trata só da aparência.

## 1. Identidade em uma frase

Alta fantasia medieval europeia pintada à mão, em vista superior 3/4, com
personagens heroicos de proporção realista sobre um cenário de vila iluminado
e saturado — quente e convidativo, nunca sombrio, apesar de o jogo ser um
survival cooperativo.

**Esta é a identidade do mundo cuidado**, e vale para tudo que a Senhora das
Trevas ainda não alcançou. A partir do mapa 2 o jogo atravessa para um bioma
que contradiz essa frase de propósito; ele tem regras próprias na **seção 11**,
e só aquela seção pode sobrepor o que está aqui. Um asset sombrio não é
licença para abandonar o resto do guia: luz, perspectiva, escala, ausência de
efeito e a relação de contraste com o personagem continuam valendo inteiras.

## 2. Técnica e acabamento

- **Pintura digital, não pixel art.** Volume por gradiente e luz, formas
  arredondadas, textura visível (veio da madeira, poros da pedra, folhas
  individuais na grama). Nunca converter para pixel art nem para vetor chapado.
- **Contorno escuro seletivo.** Os elementos de cenário têm um contorno
  marrom-escuro/quase preto (`#131B21`–`#221612`) fechando a silhueta, com
  espessura constante; os personagens não têm contorno — a silhueta deles se
  resolve por contraste de valor contra o fundo.
- **Nível de detalhe de "leitura a 1 metro".** Detalhe suficiente para
  reconhecer material e função (dobradiça, telha, pedra do rodapé), sem ruído
  que suma na escala de jogo.
- **Sem efeitos.** Nada de brilho mágico, aura, fogo, partículas ou sombra
  projetada no chão embutidos na arte. Efeitos são responsabilidade do runtime.
- **Sem fundo.** O asset é o objeto recortado; o terreno vem por baixo.

## 3. Luz

- **Direção fixa: superior-esquerda**, em todos os assets sem exceção. É o erro
  mais fácil de cometer e o mais visível quando quebra.
- Luz de dia claro, difusa, sem sol duro. Sombra própria (no objeto) suave e
  quente; nenhuma sombra projetada pintada na arte.
- Contraste médio: as áreas claras não estouram para branco puro, as escuras
  não fecham em preto puro.

## 4. Câmera e perspectiva

- **Vista superior 3/4** — o jogador vê o topo do chão e a frente dos objetos
  ao mesmo tempo. O chão é o plano XY; a altura dos objetos é sugerida
  pintando a face frontal.
- Nada de perspectiva convergente dentro de uma peça: uma cerca correndo para o
  norte é desenhada com trilhos **paralelos**, não afunilando ao fundo. Peça com
  fuga não encaixa com a vizinha (erro real cometido no kit de cerca).
- Peças horizontais e verticais são desenhadas separadamente. Girar uma
  horizontal gira também a luz, a profundidade dos postes e o contato com o chão.
- Toda peça tem uma **linha de contato com o chão** legível na base.

## 5. Paleta

Valores medidos dos assets aprovados. Uma peça nova deve cair dentro dessas
famílias, não ao lado delas.

### Terreno

| Bioma | Cores dominantes |
|---|---|
| Grama | `#5D8531` `#699032` `#759833` `#8AAA39` — verde-folha saturado, quente |
| Terra | `#AC823D` `#B88E45` `#C79E51` `#D3AD5F` — ocre quente, arenoso |
| Pedra | `#8A8676` `#9A9889` `#A8A595` `#BBB9A9` — cinza esverdeado neutro |

### Construções e madeira

| Elemento | Cores |
|---|---|
| Parede de pedra | `#A59683` claro, `#666C6C` médio, juntas escuras |
| Telhado | `#385262` azul-petróleo dessaturado — **o acento cromático da vila** |
| Madeira estrutural | `#4A3422` `#5A452C` `#6A4F32` `#7C6042` |
| Contorno/sombra | `#131B21` `#221612` |

### Vegetação

`#133323` (sombra profunda) → `#274C2B` → `#557C38` → `#74A83C` → `#BAD159`
(realce). Copas e arbustos usam a faixa inteira; folhagem solta no chão fica
nas faixas médias.

### Personagens

Os heróis são deliberadamente mais claros e dessaturados que o cenário — é
assim que o jogador se acha na tela.

| Personagem | Base |
|---|---|
| Mago | Branco-creme `#EFE3D3` `#E9DAC7`, manto azul, couro `#534336` |
| Sacerdotisa | Branco quente `#EFE6DA` `#D8C3AB`, dourado, sem armadura |
| Paladina | Creme `#F3E4CF` + ouro `#B48A5A` `#D2B187` sobre couro escuro |
| Arqueiro | Verde-floresta `#5C5030` e couro `#533A25` `#855936` — o único de valor médio-escuro |

Regra: **cenário puxa para o saturado e escuro; personagem puxa para o claro e
dessaturado.** Um asset novo de cenário nunca deve competir em claridade com um
personagem.

## 6. Escala

A grade do mundo é de **128x128 px por célula**.

| Referência | Medida |
|---|---|
| Frame do personagem | `128x192` px, silhueta ocupando 80–92% da altura |
| Personagem em tela | ~186 px de altura (`RenderScale` 1.15) |
| Porta de casa | 130–170 px de altura — mesma ordem do personagem |
| Casa pequena / média / grande | 316 / 476 / 823 px de largura |
| Poste de cerca | ~211 px de altura total, rodapé de pedra nos ~35 px de baixo |
| Árvore (tronco + copa) | ~354 px de copa sobre tronco de 140 px |

Uma peça nova se calibra contra a porta e o personagem, não contra o "tamanho
que pareceu bom". Se uma pedra fica maior que uma porta, ela virou um penhasco.

## 7. Proporção e desenho dos personagens

- Proporção **realista adulta** (~7,5–8 cabeças), não chibi, não heroica
  exagerada. Rostos com traço leve de anime ocidentalizado: olhos grandes mas
  não estilizados demais, nariz e boca discretos.
- Silhueta legível de longe: capa, manto, ombreira ou arco definem quem é o
  personagem antes de qualquer detalhe de rosto.
- **Simetria obrigatória (mirror-safe).** O renderizador espelha `W→E`,
  `SW→SE`, `NW→NE`, então nenhum detalhe pode existir só de um lado: aljava,
  espada, mecha de cabelo ou fivela assimétrica quebram ao espelhar. Detalhe
  assimétrico da referência é duplicado, centralizado ou removido.
- Vestuário de fantasia funcional: tecido pesado com dobras, couro, metal
  polido em detalhes pequenos. Ouro é acento, não superfície.
- Diversidade de aparência entre os personagens é parte do elenco (ver
  Paladina); manter.

## 8. Cenário: a vila

- Arquitetura medieval europeia rústica: alvenaria de pedra irregular com
  cantaria nos cantos, enxaimel de madeira escura, telhas de madeira/ardósia
  azul-petróleo, portas em arco de madeira maciça com ferragens pretas.
- Vida sem bagunça: vasos de flor, hera subindo pela quina, tufos de grama na
  base do muro. Detalhe pequeno e localizado, nunca cobrindo a superfície.
- Ferragens sempre em ferro preto fosco.
- A vila é **habitada e cuidada** — sem ruína, mofo, sangue ou abandono.

## 9. Checklist para um asset novo

1. Vista superior 3/4, luz vindo de cima à esquerda?
2. Escala coerente com a porta (130–170 px) e a célula de 128 px?
3. Cores dentro das famílias da seção 5?
4. Contorno escuro fechando a silhueta (cenário) / ausente (personagem)?
5. Linha de contato com o chão limpa, sem sombra ou grama pintada por baixo?
6. Sem efeito visual, sem texto, sem fundo?
7. Peças que se conectam: horizontal e vertical desenhadas separadamente, sem
   perspectiva convergente, encaixes na mesma altura?
8. Personagem: espelhável sem quebrar nenhum detalhe?

## 10. Bloco de estilo para prompts

Colar em qualquer prompt de geração de asset de cenário:

> Hand-painted high-fantasy RPG art, top-down 3/4 perspective, painterly digital
> painting with visible material texture, soft volumetric shading, light coming
> from the top-left, dark brown outline closing the silhouette, medium contrast,
> warm saturated palette (leaf green `#699032`, warm ochre `#B88E45`, neutral
> grey stone `#A8A595`, dark timber `#4A3422`, teal-blue roof `#385262`).
> Rustic medieval European village, lived-in and cared for, no ruin or decay.
> No visual effects, no glow, no cast shadow on the ground, no background,
> no text, no grid lines. Game grid is 128x128 px per ground cell; a house door
> is about 150 px tall.

Para personagens, trocar as duas últimas frases por:

> Adult realistic proportions (~7.5 heads), readable silhouette, no outline on
> the character, cream/white and gold heroic garments lighter and less saturated
> than the environment, every left/right detail symmetric so the sprite can be
> mirrored, no visual effects.

---

## 11. O bioma sombrio

A partir do mapa 2 o jogo atravessa da vila cuidada para uma mata que a Senhora
das Trevas alcançou. Esta seção é a **única** que pode contradizer as seções 1
a 10, e só no que ela declara explicitamente.

### 11.1 Identidade em uma frase

A mesma floresta, adoecida — não outra floresta, e não a mesma floresta à
noite.

### 11.2 A regra que decide tudo: corrompido ≠ noite

É o erro que este bioma vai cometer se ninguém escrever a regra:

| | Noite | Corrompido |
|---|---|---|
| Matiz | preservado, deslocado para o azul em bloco | **deslocado por material** — a folha amarela e acinzenta, a terra puxa para o frio |
| Luminância | tudo cai junto | cai, mas **cada material cai o seu tanto** |
| Contraste local | cai (some detalhe) | **preservado** — a textura continua legível |
| Saturação | cai uniformemente | sobe no que apodrece, cai no que morre |

Noite é um filtro sobre a mesma imagem. Corrupção é a imagem repintada. Se dá
para chegar no resultado escurecendo a arte clara num editor, **não é este
bioma** — é aquele bioma no escuro.

Consequência prática: a mata sombria não pode ser gerada pedindo "a versão
escura" do kit claro. Ela é um set próprio, gerado numa chamada própria.

### 11.3 Valores medidos

Medidos com o script da seção 11.7, sobre a textura inteira, em blocos de
128 px — que é uma célula de chão na tela.

| Textura | RGB médio | Matiz | Lum. | Sat. | Contraste |
|---|---|---|---|---|---|
| *(aprovado)* Grama clara | `#6F9533` | 83° | 39,2 | 48,8 | 10,3 |
| *(aprovado)* Terra | `#BD934A` | 38° | 51,5 | 46,4 | 12,9 |
| *(aprovado)* Pedra | `#A29F8F` | 51° | 59,9 | 9,3 | 18,0 |
| *(a substituir)* Grama escura | `#292C0B` | 65° | **10,8** | 59,7 | 8,1 |
| *(a substituir)* Grama rala | `#3D320F` | 46° | 14,8 | 61,5 | **6,0** |
| *(a substituir)* Terra nua | `#453725` | 34° | 20,7 | 29,8 | 11,8 |

**As três de baixo são arte de teste** (criadas em 01/08/2026 para validar o
sistema de pilhas de terreno, não para ficar). Elas estão aqui como medida do
que corrigir, não como referência a seguir.

### 11.4 Os dois defeitos medidos, e a régua que sai deles

**A grama escura está em luminância 10,8 — não é escura, é quase preta.** Ela
é quatro vezes mais escura que a grama clara. O problema não é gosto, é que
não sobra faixa embaixo dela: sombra de raiz, lado escuro de tronco, folhiço na
penumbra e a própria sombra própria dos objetos não têm para onde ir. E o
inverso é pior — sobre um chão nessa luminância, **todo prop vira fonte de
luz**. É por isso que o pinheiro dourado domina a tela no mapa 2.

**A grama rala está em contraste 6,0 — o chão lê como tinta lisa.** A grama
clara tem 10,3 e a pedra 18,0. Abaixo de ~9 a textura some na escala de jogo e
a célula de 128 px vira uma mancha de cor.

Daí a régua para a arte nova:

| Medida | Alvo | Por quê |
|---|---|---|
| Luminância do chão escuro | **18–26** | Claramente mais escuro que a grama clara (39), com faixa sobrando embaixo para sombra e para o folhiço |
| Contraste local (bloco 128 px) | **≥ 9** | Abaixo disso o chão lê como tinta lisa na escala de jogo |
| Saturação | **25–45** | Acima de ~50 a corrupção lê como fantasia colorida; abaixo de ~20 lê como preto e branco, que é noite |
| Distância de matiz do par claro | **≥ 15°**, só para vegetação | Se o matiz não se desloca, é o mesmo material escurecido — ver 11.2 |

**A regra de matiz vale para vegetação, não para solo exposto.** Terra molhada
é marrom em qualquer bioma; empurrar o matiz dela para o frio produz um chão
que parece pintado, não adoecido. O solo do bioma escuro se declara pelos
outros três eixos — cai de luminância 51 para 21, de saturação 46 para 26, e
troca o vocabulário (cascalho, raiz morta, rachadura no lugar de terra fofa de
caminho). A `terrain_bare_soil` aprovada fica a 5° da terra clara e está certa.

Na grama a regra continua estrita, e por um motivo concreto: folha tem matiz
livre para andar, então grama escura que não deslocou é grama clara com o
brilho abaixado — que é exatamente o defeito de 11.2.

### 11.4b A régua do realce: o que decide se um objeto pertence ao bioma

A média de um objeto engana. O que faz uma peça gritar na tela é o **realce** —
o percentil 90 da luminância —, e é ele que tem que ser medido.

Caso concreto, e o motivo desta seção existir. O pinheiro dourado foi gerado
para o mapa de validação, quando o chão escuro ainda era arte de teste quase
preta. Contra o terreno definitivo:

| | realce (p90) | acima do chão |
|---|---|---|
| Grama escura (o chão) | 32,2 | — |
| Carvalho grande | 37,6 | **+5,5** ✓ |
| Pinheiro dourado | 52,5 | **+20,4** ✗ |

O carvalho lê como uma árvore em pé sobre aquele chão. O pinheiro lia como uma
lanterna. E os dois têm saturação parecida — o carvalho chega a ser *mais*
saturado no p90 (100 contra 87,5). **Não era cor, era luz.**

> **Regra: um objeto do bioma sombrio fica com realce de +4 a +8 de luminância
> acima do chão em que vai ficar.** Abaixo disso ele some no terreno; acima, ele
> vira fonte de luz e denuncia que foi pintado para outro lugar.

Duas consequências práticas:

- **Consertar isso é local, não é geração nova.** O pinheiro foi corrigido por
  uma curva de ombro (`work/tiled-assets/build_forest_pine_dark.py`) que deixa o
  escuro intacto e comprime só a faixa clara, resolvida por bisseção até o p90
  cair no alvo. Multiplicar tudo por uma constante acertaria o p90 e levaria o
  tronco junto, que não era o problema.
- **A régua muda quando o chão muda.** Uma peça aprovada contra um chão de teste
  não está aprovada contra o chão final. Medir de novo custa um comando.

### 11.4c Resolução: 512 px, e por quê

Textura de terreno do bioma sombrio entra a **512 px**.

O motor recorta uma janela de 128 px por célula (`spanFor` em
`terrain_renderer.go`), então uma folha de 512 px cobre **4×4 células com
pixels 1:1**. Reduzir para 256 derruba o span para 2×2 e joga metade do detalhe
fora.

Isto também explica um erro fácil: **gerar em resolução maior não melhora nada
sozinho**. Antes do span, a folha inteira era espremida numa célula de 128 px e
perdia ~60% do contraste local; o gargalo era o destino, não a origem. O que
importa junto com a resolução é a **escala de chão**: a folha mostra ~6,4 × 6,4
metros de terreno, com a folha de grama ocupando 2–3% da largura.

### 11.5 O que **não** muda

Nada disto é negociável por ser um bioma escuro:

- **Luz superior-esquerda.** A mata é escura porque o chão é escuro, não porque
  a luz mudou de lado. Inverter a luz aqui e não no mapa 1 quebra toda peça que
  atravessa os dois.
- **Vista superior 3/4, escala da célula de 128 px, régua da porta.**
- **Sem efeito na arte.** Névoa, brilho e penumbra são runtime. Névoa pintada na
  textura fica presa no chão enquanto o jogador anda, e vira mancha.
- **Sem fundo, sem tapete.** Esta é a cláusula que já falhou uma vez: a árvore
  grande veio com um tapete de grama assado na base, na cor do próprio bioma
  escuro. Sobre grama escura quase não aparecia; sobre grama rala e terra nua
  seria um retângulo verde sob cada árvore. **Nenhum asset carrega chão junto.**
- **O personagem continua mais claro e menos saturado que o cenário.** Num chão
  de luminância 20 isso vira fácil demais: o herói passa a flutuar como um
  recorte brilhante. O alvo não é contraste máximo, é o mesmo salto de leitura
  do mapa 1 — o que faz o chão escuro precisar de luminância 18–26, e não 10.

### 11.6 Vocabulário do bioma

O que conta a história certa, e o que trai:

| Sobe | Some |
|---|---|
| Folha morta, folhiço fundo, galho seco | Flor, trevo, dente-de-leão |
| Raiz exposta, casca descascando | Grama aparada, canteiro |
| Cogumelo pálido, musgo escuro | Fruto, broto, verde novo |
| Terra rachada, solo exposto, cinza | Calçamento mantido, ferragem polida |
| Ossada, galho quebrado, tronco caído | Qualquer sinal de cuidado humano |

Duas coisas que **não** entram, apesar de parecerem do gênero: sangue e
símbolo oculto. A mata está doente, não é um cenário de ritual — e o jogo tem
personagem infantilmente legível o bastante para que sangue mude a
classificação sem mudar a leitura.

### 11.7 Como medir

```bash
python3 work/tiled-assets/measure_terrain.py assets/tilesets/<textura>.png
```

Roda sobre qualquer PNG de terreno e imprime a linha da tabela 11.3. Toda
textura nova passa por ele antes de entrar no jogo.

### 11.8 Bloco de estilo para prompts — bioma sombrio

Substitui o bloco da seção 10 em asset da mata sombria:

> Hand-painted high-fantasy RPG art, top-down 3/4 perspective, painterly digital
> painting with visible material texture, soft volumetric shading, light coming
> from the top-left, dark brown outline closing the silhouette.
> A temperate forest that has fallen sick — not a forest at night. Desaturated
> cold-shifted palette: bruised olive-green foliage going grey-yellow at the
> edges, damp dark earth, ash-grey bare wood, pale sickly mushrooms, dark moss.
> Mid-dark values with the texture still clearly readable — never near-black,
> never flat. Decay is the normal state here: dead leaf litter, exposed roots,
> peeling bark, cracked soil. No blood, no occult symbols, no ruins of masonry.
> No visual effects, no glow, no fog, no cast shadow on the ground, no background,
> no baked grass or dirt mat under the object, no text, no grid lines.
> Game grid is 128x128 px per ground cell; a house door is about 150 px tall.
