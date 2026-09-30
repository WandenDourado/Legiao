---
name: preview-skill-effects
description: Build a single-file browser preview (canvas harness) of a Legiao spell's procedural visuals, so the look can be judged and tuned WITHOUT compiling the game. Use when iterating on how a spell looks, when the agent cannot run the game itself, when the user asks "how would this look", or when growing the harness into a bench/IDE for skill effects. Companion to create-character-abilities, which decides what the spell does; this one only mirrors how it is drawn. Do NOT use for gameplay logic, network sync, sprite art, or performance verdicts.
---

# Preview Skill Effects

Port a spell's DRAW code to an HTML canvas so it can be watched, tuned and
rejected in seconds instead of in compile-run-cast cycles.

## Por que isto existe

O laco visual de uma magia custa `go build` + abrir o jogo + achar um inimigo +
conjurar + tirar print. Pior: o agente que escreve o efeito normalmente **nao
consegue ver o resultado** — ele escreve `DrawCircleGradient` no escuro e
espera o print do usuario. Foram duas rodadas as cegas no Santuario da
Sacerdotisa (31/08/2026): a primeira devolveu um miolo rosa, a segunda um
contorno que parecia o Escudo Sagrado. As duas teriam sido pegas em dez
segundos num preview.

O preview e uma **segunda implementacao do mesmo desenho**, em canvas, que roda
onde o agente e o usuario conseguem olhar juntos.

## O contrato: o preview espelha o DESENHO, nunca a jogabilidade

- Copie so das funcoes de desenho: `<spell>.go`, `<spell>_draw.go`,
  `<spell>_particles.go`. Nada de dano, colisao, rede ou host.
- As constantes vao **verbatim** do arquivo Go — leia o arquivo e copie o
  numero, nao escreva de memoria. Constante que derivou torna o preview pior
  que nenhum preview, porque ele passa a mentir com confianca.
- Mesma ordem de camadas, mesmos alfas, mesmo blending. Se voce "melhorar"
  algo no preview, ele deixou de ser um espelho.
- O que nao da para portar (shader, sprite, textura) entra como
  **placeholder declarado**, nunca como uma versao mais bonita do que o jogo
  desenha.

## Tabela de porte (raylib -> canvas 2D)

| raylib (Go) | canvas 2D |
|---|---|
| `BeginBlendMode(rl.BlendAdditive)` | `ctx.globalCompositeOperation = 'lighter'` (volte para `'source-over'` depois) |
| `rl.Fade(c, a)` | `rgba(r,g,b,a)` — canal 0..255, alfa 0..1 nos dois |
| `DrawCircleGradient(x,y,r,c,rl.Blank)` | `createRadialGradient(x,y,0,x,y,r)`, stop 0 = `rgba(c,a)`, stop 1 = **`rgba(c,0)`** (alfa a zero, NAO preto) |
| `DrawCircleV(c,r,col)` | `arc(x,y,r,0,7)` + `fill()` |
| `DrawRing(c, inner, outer, 0, 360, seg, col)` | `arc` com `lineWidth = outer-inner` e raio `(inner+outer)/2` + `stroke()` |
| `DrawRing(...)` com arco parcial | `arc(x,y,r, start*PI/180, end*PI/180)` — **raylib usa graus** |
| `DrawLineV` / `DrawLineEx` | `moveTo/lineTo` + `lineWidth` |
| `DrawRectangleGradientV` | `createLinearGradient` vertical |
| `DrawEllipse` | `ctx.ellipse` |
| `DrawPoly` / `DrawTriangle` | `beginPath` + vertices |

Armadilhas:

- **Graus x radianos.** Todo angulo de raylib e grau; canvas e radiano.
- **Y cresce para baixo nos dois** — nao inverta nada.
- `'lighter'` nao e identico ao `BlendAdditive` (premultiplicacao), mas erra
  para o mesmo lado: se o preview esta lavado, o jogo tambem estara.
- Gradiente de raylib vai para `Blank` (alfa 0). Terminar em preto no canvas
  cria um halo escuro que o jogo nao tem.

## O esqueleto do harness

Um arquivo `.html`, tudo inline, sem import e sem CDN — ele precisa abrir
direto do cartao de arquivo do chat.

1. **Constantes do jogo no topo**, com o nome que tem no Go (`R`,
   `GROUND_DROP`, `DUR`, `FADE`, cores da paleta).
2. **Escala e canvas.** ~600 px de lado e um `S` (1,2–1,4) para o raio real
   caber. Guarde a escala numa constante: todo raio desenhado passa por ela.
3. **Estado espelhando os campos do FX struct** (`time`, `waves`, `motes`...),
   com as mesmas funcoes de atualizacao.
4. **Laco `frame(now)`** com `dt` limitado a 0,05 s — o jogo roda com dt
   variavel, e um preview com dt de aba em segundo plano da saltos.
5. **Fundo igual ao lugar onde a magia e julgada.** Grama verde para o campo,
   escuro para o bioma sombrio. Um efeito julgado sobre fundo preto MENTE: o
   miolo rosa do Santuario era invisivel no escuro e obvio sobre a grama.
6. **Figura placeholder na ancora**, na proporcao certa (sprite 128x192 com
   `RenderScale` 1.15). Sem ela nao da para julgar o tamanho da aura em
   relacao ao personagem.
7. **Ciclo de vida em laco**: duracao + fade + reset. A saida faz parte do
   design; um preview que so mostra o auge esconde metade dos defeitos.
8. **Movimento do conjurador** (um checkbox "andando") sempre que o efeito for
   ancorado em alguem. Aura que segue tem de ser julgada em movimento.
9. **Sliders so para os 1–3 numeros em discussao**, imprimindo o valor ao
   lado. Todo o resto fica literal: cada slider a mais e um lugar onde o
   preview pode divergir do codigo.

## O que o preview prova, e o que ele nao prova

**Prova:** matiz e temperatura sobre o fundo real; silhueta e leitura de
relance (teste do olho semicerrado); ordem das camadas e empilhamento de alfa;
ritmo e cadencia; proporcao em relacao ao personagem; se o efeito **resolve**
ou se ele some de estalo; e — o que salvou o Santuario — se a magia parece de
outra classe, quando se poe as duas lado a lado.

**Nao prova:** fps, fill rate, custo de GPU, culling, raio real de
jogabilidade, ordem de desenho contra o mundo, oclusao por sprite, tempo de
rede. Nada disso e portavel para o canvas.

**Por isso:** nunca escreva "o desempenho esta bom" a partir de um preview. O
juiz de custo continua sendo o F3 no mapa mais pesado
(`doc/skill_performance.md`), e o preview nao substitui uma unica linha
daquele checklist.

## Entrega

- Escreva em `/tmp` (ou no diretorio de trabalho do agente), **nunca** dentro
  das pastas do usuario sem ele pedir.
- Entregue o arquivo para renderizar no chat.
- Diga em uma linha **quais numeros sao ajustaveis** e que fps nao se julga
  ali.
- Quando o usuario escolher os valores, edite as constantes no Go e repita.

## Se isto crescer para uma IDE de efeitos

O passo de um preview para uma bancada e pequeno, e vale a pena na terceira
magia:

- **Registro em vez de arquivo por magia.** Cada efeito vira
  `{ id, consts, state(), update(dt, st), draw(ctx, st, alpha) }`; um seletor
  troca qual esta ativo. Um canvas so, um laco so — as magias diferem apenas
  nessas tres funcoes. E a mesma forma do `ability.Registry` do jogo, o que
  mantem os dois mundos pensando igual.
- **Scrubber de tempo** e o recurso mais valioso de todos, mais que qualquer
  slider: julgar um estilhaco de 0,45 s exige **parar dentro dele**. Pausa e
  avanco quadro a quadro vem junto.
- **Swatches de fundo** por mapa (grama, pedra, bioma sombrio, arena da
  chefe), porque a cor certa depende do chao.
- **Modo "duas magias ao mesmo tempo"**, lado a lado: e o unico jeito de
  perceber que duas classes ganharam a mesma gramatica visual.
- **Botao "copiar constantes"** que emite um bloco `const (...)` em Go pronto
  para colar — o caminho de volta fecha o laco.
- **Ler as constantes do Go, nao copia-las.** No dia em que a bancada virar
  ferramenta de verdade, ela deve parsear `internal/skill/<spell>.go` (as
  constantes sao literais nomeadas, num bloco `const`, justamente por isso).
  Constante copiada a mao envelhece, e uma bancada desatualizada e pior que
  nenhuma.
- A bancada **nao e o jogo**, e precisa dizer isso na propria tela: uma linha
  fixa lembrando que fps e culling se medem no F3.

## Exemplo pronto

`references/santuario.html` — o Q da Sacerdotisa: poca de luz no chao, ondas
que nascem nela e morrem no alcance da cura, motes que acendem quando a onda
passa (`waveBoost`), tudo aditivo sobre grama, com checkbox de caminhada e dois
sliders de cadencia. Portado de `internal/skill/sanctuary_draw.go` e
`sanctuary_particles.go`. Use como molde: troque as constantes do topo, as tres
funcoes de desenho e o fundo.

## Definition of done

- [ ] Constantes conferidas **contra o arquivo Go**, uma a uma.
- [ ] Fundo igual ao lugar onde a magia aparece no jogo.
- [ ] Roda em laco (duracao + fade + reset) sem recarregar a pagina.
- [ ] O conjurador pode se mover, se o efeito o acompanha.
- [ ] Sliders so nos numeros em discussao.
- [ ] Um arquivo, zero requisicoes externas.
- [ ] Entregue dizendo o que ajustar — e que fps nao se mede ali.
