# Diálogo

Cenas narrativas que param o jogo. O host conta a história, os clientes leem.

## Modelo

- O host é o narrador: só ele dispara uma cena, só ele avança de linha e só ele encerra.
- Cliente é display: recebe a linha atual e desenha. Não avança, não pula, não dispara nada.
- Enquanto a cena roda, o mundo congela para todos (nenhum input, nenhuma simulação).
- Jogo solo (sem rede) funciona igual: o próprio jogador é o narrador.

## Onde ficam os roteiros

Um arquivo JSON por mapa, com o nome do mapa:

| Mapa | Roteiro |
|---|---|
| `assets/maps/world_01.json` | `assets/dialogues/world_01.json` |

Mapa sem arquivo é mapa silencioso — não é erro, o loader só registra no log.

### A campanha hoje (narrativa de 10/09/2026)

Todos os sete mapas têm roteiro. A tabela diz qual cena existe em cada um e
quem é o herói do resgate — a última fala do clímax é a deixa dele
(`ResolveLastStand`), por isso ela nunca pode sair do fim do roteiro.

| Mapa | Abertura (`on_map_start`) | Clímax (`on_last_stand`) | Fim de fase (`on_waves_cleared`) |
|---|---|---|---|
| `world_01` vilarejo | `world_01_abertura` | — | `world_01_hordas_limpas` |
| `world_02` floresta | `world_02_abertura` | `world_02_climax` — "Ergam-se." (Necromante) | `world_02_hordas_limpas` |
| `world_03` fortaleza | `world_03_abertura` | `world_03_climax` — "IRMÃOS, RECEBAM A LUZ DO DIVINO!" (Sacerdotisa) | `world_03_hordas_limpas` |
| `world_04` saguão | `world_04_abertura` | `world_04_climax` — prece do Arqueiro | `world_04_saida` |
| `world_05` arena | `world_05_abertura` | `world_05_climax` — "Que os céus caiam sobre eles!" (Mago) | `world_05_hordas_limpas` |
| `world_06` corredor | `world_06_abertura` | `world_06_climax` — "Eu sou a Avatar dos deuses!" (Paladina) | `world_06_canhoes_destruidos` |
| `world_07` trono | `world_07_abertura` | — | `world_07_vitoria` (libera fogos, "Missão Concluída" e a música) |

A antagonista se chama **Imperatriz das Trevas** em todo texto visível (barra
da chefe, anúncio da horda, tela de vitória). Os nomes internos continuam os
antigos — `EnemyTypeDarkLady`, a pasta `senhora_das_trevas` — porque não
aparecem para o jogador.

As falas usam só caracteres do Latin-1 (acentos, cedilha, `À`, `Ã`): é o que a
fonte padrão do raylib desenha. Travessão (`—`) e reticências de um caractere
(`…`) saem como `?` — escreva `...` com três pontos.

```json
{
  "map": "assets/maps/world_01.json",
  "scripts": [
    {
      "id": "world_01_abertura",
      "trigger": "on_map_start",
      "lines": [
        { "speaker": "Paladina", "portrait": "paladina", "text": "..." },
        { "speaker": "", "portrait": "", "text": "A neblina cobre a estrada." }
      ]
    }
  ]
}
```

| Campo | Regra |
|---|---|
| `id` | Único no jogo inteiro, não só no mapa. É o que marca a cena como já vista. |
| `trigger` | `on_map_start`, `on_last_stand` ou `on_waves_cleared`. Um roteiro por gatilho, por mapa. |
| `speaker` | Nome exibido. Vazio = narração sem nome (o "Narrador" do texto da narrativa entra assim). |
| `portrait` | Chave de personagem (`mago`, `paladina`, `sacerdotisa`, `arqueiro`, `necromante`). Vazio = sem imagem. |
| `text` | A fala. Quebra de linha é automática, escreva o parágrafo inteiro. |

Editar texto não exige recompilar: o arquivo é lido em runtime, como os mapas.

## Gatilhos

| Gatilho | Quando dispara |
|---|---|
| `on_map_start` | Primeiro quadro em que o mapa está no ar (inclusive depois de um portal). |
| `on_waves_cleared` | Quando `WaveState.Phase` vira `cleared`, ou seja, a última horda do mapa morreu. No corredor dos canhões (`world_06`), que não tem horda, quando o último canhão cai. |

Cada roteiro toca uma vez por sessão, guardado pelo `id`. Voltar para um mapa não repete a cena.

## Retratos

Cada personagem tem **duas** artes, e elas respondem a perguntas diferentes:

| | Tela de seleção | Caixa de diálogo |
|---|---|---|
| Arquivo | `assets/sprites/<char>/reference.png` | `assets/sprites/<char>/portrait.png` |
| Campo | `CharacterDef.ReferenceImagePath` | `CharacterDef.PortraitImagePath` |
| Tela | 1024x1536 RGBA | 1152x1280 RGBA |
| Enquadramento | corpo inteiro, pé em `y = 1470` | meio-corpo, cortado na coxa |
| Pergunta que responde | "que classe é essa?" | "quem está falando, e com que cara?" |
| Leitura | silhueta | **rosto** |

Continua valendo o que valia antes: o roteiro cita uma **chave de personagem**
(`"portrait": "mago"`), nunca um caminho de imagem. Trocar a arte de um
personagem não reescreve roteiro nenhum.

**Se `portrait.png` faltar, o retrato cai para a `reference.png`** com o
recorte antigo (`internal/ui/portrait_cache.go`, `loadPortrait`). A queda existe
por ordem de trabalho — o código desta caixa entrou antes da arte —, não como
comportamento desejado: arte de corpo inteiro desenhada no retângulo do retrato
sai com o rosto pequeno e o corpo cortado no lugar errado.

Três casos desenham **sem** imagem nenhuma:

1. `portrait` vazio — narração, voz de fora, explicação.
2. Chave que não existe no registro de personagens (erro de digitação). Sem essa
   checagem, `entity.GetCharacter` devolveria o Mago como fallback e a fala
   sairia com o rosto errado.
3. Personagem listado em `placeholderPortraits` (`internal/ui/dialogue_box.go`) —
   arte ainda emprestada de outro personagem. Hoje a lista está **vazia**.

### Como o retrato e a caixa se posicionam

`internal/ui/dialogue_geometry.go` é a fonte única: `dialogueLayout` devolve os
dois retângulos, e `DialogueBoxRect` — o alvo de toque — sai dela. Antes a caixa
e a moldura do retrato eram calculadas em lugares diferentes, e só um deles era
o alvo de toque.

O retrato **deriva** da caixa. `portraitAboveBox = 0.58` é o único número que
decide onde a caixa corta o personagem, e é a derivação que o faz valer em
qualquer proporção de tela — 16:9, celular deitado, 4:3 ou 4K.

Duas escolhas que não são estéticas:

- **O retrato é desenhado ENTRE o escurecimento e o painel.** É essa ordem que
  produz a leitura pretendida: o busto sobe livre acima da caixa e o painel
  cruza por cima do quadril. Desenhar o painel antes devolveria a figurinha
  colada ao lado da caixa.
- **A âncora do retrato é a linha de topo da caixa, não a borda de baixo da
  tela.** Sem a trava de largura as duas dão o mesmo número. Com ela ativa
  divergem, e a diferença decide se existe cena: ancorado na borda de baixo, um
  retrato encolhido numa tela em pé cabia inteiro abaixo do topo da caixa e o
  orador sumia atrás do painel.

`internal/ui/dialogue_geometry_test.go` trava essas invariantes em seis
proporções de tela.

## Avançar

| Plataforma | Comando (só no host) |
|---|---|
| Desktop | `Enter` ou `Espaço` |
| Android/toque | Toque **dentro da caixa** de diálogo |

O toque só conta dentro da caixa (`ui.DialogueBoxRect`) para que um toque no joystick não pule metade da cena. Avançar na última linha encerra a cena — não existe botão de fechar separado.

O quadro que encerra a cena ainda é consumido pelo diálogo, então o clique que fecha a caixa não dispara um ataque logo em seguida.

## Congelamento

O loop devolve o quadro inteiro ao diálogo (`internal/game/loop.go`): quando `DialogueDirector.Update` retorna `true`, só câmera e desenho rodam. Ficam parados:

- input de movimento, ataque e habilidade (host e cliente);
- `Host.UpdateSimulation` — inimigos, projéteis, combate, hordas, cooldowns;
- animação de efeitos no cliente;
- timer de respawn e a transição de portal.

Além disso o host **descarta** `attack` e `skill` recebidos enquanto a cena roda, para o caso de um cliente que ainda não recebeu a pausa.

## Rede

| Mensagem | Direção | Conteúdo |
|---|---|---|
| `dialogue` | host -> peers | `DialogueState`: ativo, roteiro, falante, retrato, texto, posição na cena. |

O texto trafega inteiro em vez de um id de linha: assim o cliente não precisa do arquivo de roteiro e uma pasta `assets` desatualizada no cliente não mostra uma fala diferente da que o host está narrando.

O envio é por mudança, não por quadro, porque o broadcast periódico do host mora dentro de `UpdateSimulation` — exatamente o que a cena congela. Quem entra (ou volta) recebe o estado na hora do join (`sendDialogueTo`).

**O rejoin sincroniza o estado do diálogo, inclusive a AUSÊNCIA de cena**, e essa palavra é o contrato. `sendDialogueTo` tinha um `if !d.Active { return }`, escrito para o caso de quem entra no meio de uma cena; o caso que faltava era o inverso. `Active:false` é publicado **uma vez**, como evento, por `DialogueDirector.end` — quem estava com a conexão caída naquele instante nunca recebe, e o host ficava calado justamente no rejoin. O cliente guardava para sempre o `Active:true` de antes da queda e, como `DialogueActive` é o portão de pausa de input e simulação, a pessoa voltava para um jogo congelado numa caixa que só o host podia avançar — e o host já tinha encerrado a cena. (Relato de uma testadora, 20/09/2026.)

O par disso está no cliente: `triggerReconnect` (`internal/network/reconnect.go`) **zera** `CurrentDialogue` ao perceber a queda. Enquanto a conexão está de pé, o estado da cena é do host; caída, o que sobra na máquina do cliente é uma cena que ela não tem como avançar nem encerrar. Durante a tentativa quem fala é o overlay de reconexão, e o que o host tiver a dizer sobre a cena chega no rejoin.

Cobertura: `internal/network/dialogue_rejoin_test.go`.

## Arquivos

| Arquivo | Responsabilidade |
|---|---|
| `internal/dialogue/script.go` | Tipos do roteiro (dado puro, sem rede e sem desenho). |
| `internal/dialogue/load.go` | Leitura e validação do JSON do mapa. |
| `internal/dialogue/runner.go` | Cursor sobre uma cena em execução. |
| `internal/game/dialogue.go` | Quem dispara, quem avança, quem encerra (host/solo). |
| `internal/game/dialogue_stage_cleared.go` | `stageFightOver`: a condição de `on_waves_cleared` (horda limpa, ou canhões calados). |
| `internal/network/dialogue.go` | Estado compartilhado e publicação para os peers. |
| `internal/ui/dialogue_box.go` | Caixa, retrato, quebra de texto, dica de comando. |

## Limitações conhecidas

- Um roteiro por gatilho por mapa. Cena maior = mais linhas, não mais roteiros.
- Não há escolha de resposta nem ramificação: a cena é linear.
- Não há efeito de máquina de escrever; a linha aparece inteira.
- Não há título sob o nome do orador (o "Prince of the Underworld" do Hades).
  Seria um campo no registro de personagem, não no roteiro: nenhum JSON mudaria.
- O texto usa a fonte padrão do raylib. Se algum acento aparecer como bloco no jogo, a saída é embutir uma fonte com `LoadFontEx` — não tirar os acentos do JSON.


## Gatilhos

| Gatilho | Quando | Da corrida? |
|---|---|---|
| `on_map_start` | Primeiro quadro do mapa no ar | não (é do mapa) |
| `on_last_stand` | **Durante** a luta, quando o grupo está caindo | sim |
| `on_waves_cleared` | Última horda morta — ou último canhão caído, no mapa sem horda (`stageFightOver`) | sim |

`knownTrigger()` em `load.go` é a lista única de aceitos: gatilho que o diretor
não sabe responder vira roteiro que **nunca toca, em silêncio**.

### `on_last_stand`

O único que dispara durante a luta. Existe para o momento em que um mapa é
desenhado para ser perdido, de modo que a cena aconteça **em vez** do Game Over.
Condição e consequências em `doc/combat_rules.md`.

**Ele exige uma janela declarada.** `partyIsFalling` pergunta a
`network.ClimaxWindowOpen(mapPath, zones)`, que lê `internal/network/
climax_window.go` (`climaxWindows`) — a mesma FASE DECLARA, RESTO LÊ que
`waveRuns`, `climaxRuns` e `lastStandHeroes` já usam. Um mapa sem entrada
nessa tabela nunca chega a avaliar o gatilho: o roteiro existe e fica em
silêncio. Diferente de um esquecimento comum, este é logado — se o mapa TEM
um roteiro `on_last_stand` e NÃO tem janela declarada, `syncMap` avisa no log
(`[Climax] ... roteiro on_last_stand mas nenhuma janela ...`), o mesmo
cuidado que `StartWaveRun` já tem para marcadores sem corrida.

A janela em si tem três naturezas — por horda (a partir de qual), emboscada
(enquanto a corrida do clímax está no ar) ou checkpoint (zona alcançada) —,
ver a tabela completa em `doc/combat_rules.md`.

Quem se ergue no resgate é **do mapa**, não do jogo: ver a tabela
`lastStandHeroes` em `internal/network/last_stand_heroes.go`.

### Cena da corrida × cena do mapa (`Trigger.PerRun`)

O `played` do diretor é por id de roteiro, e ele **esquece as cenas da corrida**
quando a fase reinicia (`network.StageGeneration()`).

Isto já foi um bug: o roteiro do clímax seguia marcado como tocado depois de um
F5, a cena não abria na segunda tentativa, e como é ela que segura o Game Over
o grupo perdia direto.

A abertura **não** é esquecida — reiniciar a fase não devolve o grupo à
floresta, e repetir a conversa de chegada a cada tentativa só cansaria.

**Ao criar uma cena nova, decida a qual das duas famílias ela pertence.**
