# Plano: bots de classe

> **Estado: planejamento. Nada implementado.** Este documento decide *o quê* e
> *por quê*; os números são pontos de partida para afinar em jogo, e a razão de
> cada um é o que impede a afinação de desfazer o desenho.
>
> Decisões tomadas com Gui em 20/08/2026. Revisado no mesmo dia contra o código
> (ver §13, *Contrato de execução*, e §14, *Armadilhas verificadas no código*):
> quem for implementar deve ler as duas seções finais antes de escrever a
> primeira linha.

O jogo foi feito para cinco. Hoje, quando o grupo não se completa, as classes
que faltam simplesmente não existem — e as hordas, que foram *escritas para
cinco* (`doc/plan_dificuldade_5_jogadores.md`), medem outra coisa. Este plano
preenche as classes vagas com bots.

---

## 1. As decisões

| Pergunta | Decisão |
|---|---|
| Quem ganha bot | Toda classe sem humano. O host escolhe Mago → as outras quatro são bots. |
| Cliente entra com classe que tem bot | **Assume o bot**: o corpo continua em campo, o dono muda. |
| Cliente entra com classe que já tem humano | Nasce um personagem **novo** daquela classe. O grupo passa de cinco, e isso é aceito. |
| Bot conta como membro do grupo | **Em tudo**: Game Over, cena do último suspiro, porta do portal, contagem do HUD. |
| Cena do último suspiro | O **bot** da classe faz o resgate. O NPC (`game/last_stand_npc.go` + `Host.summonHeroNPC`) vira saída de emergência. |
| Nível de jogo | Como um humano experiente. Nada de bot burro — e nada de reação instantânea, que lê como trapaça. |
| Supremas | Só quando a narrativa liberar, pelo **mesmo** portão do jogador humano. |
| Cor e rótulo do bot | Cor vazia (`Color: ""`), como o NPC do último suspiro: a arte da classe é a identidade. **Sem rótulo "bot" na tela nesta entrega** — ver §12. |

---

## 2. O princípio: um bot é um jogador do host

A frase que decide todo o resto: **um bot é uma entrada em `h.players`, não uma
criatura nova.**

O que isso entrega de graça, sem uma linha escrita para bot nenhum:

- `state_update`, identidade anunciada pelo roster, interpolação no cliente,
  desenho pelo caminho de jogador remoto, barra de vida do aliado;
- morte, contagem de ressurreição e revive (`host_death.go`);
- `PlaceEveryoneAtSpawn` na travessia de portal e o reinício de fase;
- cooldowns por `playerID`, portão de cadência de ataque, portão de suprema;
- alvo de monstro (`getAllPlayersForAI` → `FindNearestPlayer`), cura da
  Sacerdotisa que atravessa aliados, `checkGameOver`, `PresentPlayers`.

E o custo: **nenhuma mensagem nova de protocolo e nenhuma mudança no cliente.**

**Contra-desenho recusado:** o bot como `entity.Enemy` aliado, ou como uma
terceira categoria com desenho próprio. Repetiria tudo da lista acima e traria
de volta, em cada arquivo, a pergunta que este repositório já respondeu uma vez
por todas: *isto conta como jogador?*

---

## 3. A vaga é da classe; o dono muda

O identificador do bot é sintético: `bot_<classe>` — o mesmo formato de
`npc_<classe>` que o último suspiro já usa.

**Uma função reconcilia, e ela é chamada de quatro lugares.**
`Host.ReconcileBots()` (exportada: `game` a chama) percorre
`entity.AllCharacters()` e garante o invariante:

> Uma classe tem exatamente um bot quando nenhum humano a joga; nenhum bot
> quando algum humano a joga.

- **Humano ausente conta como humano.** Senão a tela bloqueada de cinco minutos
  (`ReconnectGrace`) faria um bot nascer ao lado do corpo parado do jogador, e
  os dois brigariam pela vaga no rejoin.
- Os quatro pontos de chamada, e só eles:
  1. `World.ApplyToHost` (`game/world_state.go`) — carregamento e troca de mapa;
  2. `Host.handleJoin` (`network/host_rejoin.go`) — entrada e reconexão;
  3. `Host.tickAbsence` (`network/host_absence.go`) — remoção definitiva;
  4. `Host.ResetStage` (`network/host_reset.go`) — reinício de fase (F5).
- **Toda criação ou remoção de bot termina em `h.BroadcastRoster()`.** É o
  roster que carrega identidade: o snapshot de tique passa por `slimPlayer`
  (`wire.go`), que apaga `Character`, `Color` e `MaxHealth`. Um bot anunciado só
  pelo tique chega ao cliente sem personagem, e `entity.GetCharacter` cai no
  Mago — quatro Magos na tela.
- **Ordem em `ApplyToHost`:** reconciliar *depois* de `host.Skills.Reset()` e da
  instalação de guarnição/sentinelas. Em `world_travel.go`, `ApplyToHost` roda
  antes de `PlaceEveryoneAtSpawn`, que move *todos* os jogadores — os bots
  recém-criados inclusive. É a ordem certa; não inverter.

### Tomada de posse

Cliente entra com a classe de um bot: o humano **herda o corpo** — posição,
vida, morte e contagem de ressurreição, cooldowns — e o bot deixa de existir.
É uma troca de chave, não um nascimento.

- Migrar os mapas por jogador: `skillCooldowns` e `skillCharges` (chave
  `playerID|skillID` — reescrever o prefixo), `attackCooldowns`,
  `sanctuaryCooldowns`, invulnerabilidade (`host_invulnerability.go`),
  `testPlayers`.
- **Apagar os efeitos no ar do bot** (`Skills.ClearOwner("bot_x")`, §4) e
  **transmitir o sinal de fim de cada efeito apagado** (`legion_end`,
  `angelic_end`, `avatar_end`, ...) — o cliente desenha esses efeitos até
  receber o sinal, e um dono que sumiu nunca mais o envia.
- Por que herdar a recarga e não os feitiços: herdar o corpo é continuidade;
  herdar a suprema recarregada seria um botão de reset — entre com a classe,
  saia, entre de novo.
- Depois: `BroadcastRoster`, `sendStateToClient`, `sendUltimateGrantsTo` — o
  caminho de `handleJoin` já faz os três.

### Saída de humano

Ausência **não** devolve a vaga: são cinco minutos de graça, e o bot que
nascesse no meio deles teria de sumir no rejoin. Quando `tickAbsence` remove o
slot de vez, `ReconcileBots` cria o bot **herdando o corpo do removido** — a
operação simétrica da posse, pela mesma razão.

### Classe duplicada

O segundo humano da mesma classe é um jogador comum, com o próprio `PlayerID` e
o spawn que o join já dá. Nenhum bot é removido: o grupo fica com seis, e a
régua "escrita para cinco" passa a medir a menos. **Limite conhecido e aceito**
— registrar em `doc/combat_rules.md` e revisitar se a campanha ficar fácil.

---

## 4. Onde o código toca

### Fase 0 — o defeito que precisa cair antes (obrigatório)

**`RemotePlayers` nunca é podado no host.** `RemovePlayerState`
(`network/globals.go`) não tem um único chamador, e nem `BroadcastRoster` nem
`BroadcastStateUpdate` apagam quem saiu de `h.players`: os dois só escrevem por
cima. Hoje isso é um fantasma raro (jogador removido por `tickAbsence` continua
desenhado e continua contando em `PresentPlayers`). **Com bots vira rotina**:
toda tomada de posse remove um `bot_<classe>`, e o fantasma dele seguraria a
porta do portal para sempre — `game/portal_party.go` conta a festa por
`PresentPlayers`.

Correção: em `BroadcastStateUpdate` e `BroadcastRoster`, no papel host, apagar
de `RemotePlayers` todo ID que não está mais em `h.players`. Um teste cobre
isso (§11).

### Pacote novo: `internal/bot`

**Não importa `network`** — a dependência é `network → bot`; o contrário fecha
um ciclo. Consequência prática: tudo o que a IA precisa saber sobre progressão,
recarga e portal chega dentro da `View`, preenchida pelo host. Nenhuma chamada
a `network.UltimateUnlockedFor` de dentro do pacote `bot`.

| Arquivo | Responsabilidade |
|---|---|
| `bot/view.go` | `View`: o que o bot enxerga (§13.2). |
| `bot/intent.go` | `Intent`: direção de movimento, mira do ataque, magia a lançar. Nada mais. |
| `bot/brain.go` | `Brain interface { Think(View) Intent }` e `For(char) Brain`. |
| `bot/steering.go` | Aproximar, kitar, ficar *entre* A e B, separar-se dos aliados, contornar. |
| `bot/tuning.go` | Todos os relógios e limiares, num lugar só, comentados (§13.4). |
| `bot/paladina.go`, `sacerdotisa.go`, `arqueiro.go`, `mago.go`, `necromante.go` | Uma classe por arquivo. |

### `internal/network`

| Arquivo | O que faz |
|---|---|
| `host_bots.go` (novo) | `ReconcileBots`, criação/remoção da vaga, `h.bots map[string]*botRuntime` (§13.3). |
| `host_bot_takeover.go` (novo) | Posse e herança: migração de recargas, `ClearOwner` + sinais de fim, herança na saída. |
| `host_bot_tick.go` (novo) | `tickBots(dt)`, chamado de `UpdateSimulation`: monta a `View`, chama o `Brain`, aplica o `Intent`, **publica os bots em `RemotePlayers` a cada quadro** (§14.3). |
| `host_bot_move.go` (novo) | Movimento resolvido contra `EntityManager.Solid` — o resolvedor do jogador — e animação sem GPU. |
| `host_bot_portal.go` (novo) | `SetPartyPortals(rects []rl.Rectangle, active bool)`: o único caminho pelo qual o host aprende que portais existem e onde (§14.1). |
| `host.go` | `BroadcastStateUpdate` poda `RemotePlayers` (fase 0). |
| `host_rejoin.go` | `handleJoin` consulta a vaga antes de criar jogador novo; reconcilia ao final. |
| `host_absence.go` | `tickAbsence` reconcilia depois de remover. |
| `host_reset.go` | `ResetStage` reconcilia depois de repor o campo. |
| `host_last_stand.go` | `reviveHero` prefere humano a bot (§7). |

### `internal/entity`

- Expor `WalkRowFor(dir rl.Vector2) int` (hoje `walkRowForDirection`).
- `player_anim_headless.go` (novo): avanço de quadro/linha **puro**, sem
  textura e sem GPU. O bot publica `CurrentFrame`, `CurrentRow`, `VelX` e
  `VelY` como qualquer jogador, ou desliza pelo mapa em pose de parado.
- Nada de `MoveByGroundCorrection` para bot: ela recebe `*Player`, e o bot é um
  `PlayerState`. Use `GroundBoxAt(pos, char)` + `collision.Resolve` e aplique a
  diferença (`resolved - wanted`) à posição — a mesma conta, sem `*Player`.

### `internal/skill`

`clear_owner.go` (novo): `Manager.ClearOwner(ownerID string) []string`, devolvendo
os sinais de fim que precisam ser transmitidos.

**Atenção — as coleções têm duas chaves diferentes**, e um `delete(m.X, owner)`
uniforme seria um no-op silencioso em metade delas:

| Chaveadas por **dono** | Chaveadas por **ID do efeito** (têm campo `OwnerID`) |
|---|---|
| `Shields`, `Avatars`, `Angelics`, `Legions` | `Fireballs`, `Arrows`, `Sanctuaries`, `Swords`, `Meteors`, `Celestials`, `Graveyards` |

Cada grupo sob o próprio mutex, no mesmo desenho de `Manager.Reset()`
(`skill/reset.go`) — e não um `Manager` novo.

### `internal/game`

- `world_state.go` (`ApplyToHost`): `host.ReconcileBots()` (ordem no §3).
- `portal.go` (`UpdatePortal`): no papel host, informar o portal ativo por
  `SetPartyPortals` (§14.1). É a única linha de portal que a rede vê.
- **Nada em `loop.go`** — `AGENTS.md` proíbe lógica de jogo ali; a chamada de
  bot entra em `UpdateSimulation`, que o laço já invoca.

### `internal/system`

Não usar. O pacote não tem um único importador (`combat.go`, `movement.go`,
`spawn.go` são código morto) e parece o lugar certo para IA de bot — não é.

---

## 5. A inteligência

### O esqueleto

Percepção → decisão → atuação, todo quadro, mas com **relógios**:

- **Alvo é reavaliado a cada ~250 ms**, não a cada quadro. Sem isso o bot
  alterna entre dois inimigos equidistantes e vibra no lugar.
- **Reação de 200–300 ms** a fatos novos (um inimigo entra no alcance, um aliado
  cruza o limiar de vida). Isto não é burrice fabricada: é a diferença entre um
  companheiro e um script. Responder no quadro exato lê como trapaça.
- **Mira prevê o alvo** pela velocidade dele, porque os projéteis viajam. Sem
  erro artificial somado: o pedido é que ele jogue bem.

**A atuação passa pelos mesmos portões do humano:** `h.HandleAttack` e
`h.HandleSkillMessage`. Nada de caminho paralelo — cadência, recarga, cargas,
suprema travada e diálogo em cena já são decididos ali, e um segundo caminho
seria um segundo conjunto de regras para divergir no primeiro conserto.

### Movimento, para todos

- Colisão pelo resolvedor do jogador (`collision.Resolve` contra
  `EntityManager.Solid`).
- **Separação entre aliados.** Cinco personagens no mesmo pixel viram um borrão
  e morrem do mesmo golpe em área. Isto não é enfeite.
- **Seguir o grupo, mas o líder é humano.** O destino de "seguir" nunca é a
  média de todos os vivos (`PartyCentre`) — um bot adiantado puxaria a própria
  referência de avanço junto com ele, e o resto seguiria o centro que
  acabou de andar. É `HumanCentre`, a média só dos humanos vivos; sem humano
  vivo, o bot segura posição e luta em vez de escolher um novo líder entre os
  bots. E o ponto não é o `HumanCentre` cru: é `formationPost`, o posto da
  classe relativo a ele na direção do avanço (`View.AdvanceDir`, a
  velocidade média dos humanos suavizada, mantida quando o grupo para) —
  Paladina 160px à frente, Arqueiro/Mago 250px atrás em lados opostos,
  Necromante 250px atrás mais afastado, Sacerdotisa 350px atrás. Só sai
  do posto passado `followRadius` de distância dele. Ver
  `doc/plan_avanco_bots_e_gargula.md` §A2/§A3.
- **Um inimigo fora de `engageRadius` (900px) do bot OU do `HumanCentre` não
  existe para a decisão.** `engageableFoes` filtra antes de qualquer escolha
  de alvo — é a diferença entre "há um monstro no mapa" e "há um monstro no
  meu caminho", e é o que impede o bot de atravessar o mapa 3 até o clímax
  ignorando a guarnição pelo caminho.
- **Recuo com histerese abaixo de 35% de vida, só volta a engajar acima de
  60%** (`retreatHysteresis`) — sem a folga entre os dois limiares, um único
  golpe cruzando a linha faz o bot vibrar entre lutar e fugir. Recuando, o
  destino é o posto de formação empurrado 300px mais para trás, na direção
  oposta ao inimigo mais próximo (`retreatDest`). Arqueiro, Mago, Necromante e
  Sacerdotisa continuam atirando enquanto recuam — só a Paladina para de
  golpear, e só recua abaixo de 25% E com o Escudo já gasto (senão a linha da
  frente abandonaria o grupo na primeira queda de vida em vez de tentar
  mitigar primeiro).
- **Ataque básico só sai dentro do alcance real do próprio projétil**
  (`arqueiroAttackRange`, `magoAttackRange`, `necromanteAttackRange`, e
  `boltRange` da Sacerdotisa) — sem isso o Arqueiro gastava a cadência em
  flechas que expiravam a caminho de um alvo a mais de 1120px.
- **Porta do portal**: quando o host declara um portal ativo, o destino do bot
  é o retângulo alvo (`bot.SeekPortal`, sem separação — ver abaixo). Sem isso a
  porta nunca abre — porque o bot conta.
- **Quem entra no portal some e espera.** Contar "todos dentro ao mesmo tempo"
  não sobrevivia a um bot entrando e saindo do retângulo o tempo todo, então a
  regra mudou: quem pisa na zona do portal (humano ou bot) SOME da tela e
  CONGELA, liberando o pequeno retângulo para o resto do grupo. Um bot congela
  porque `tickBots` pula por inteiro quem está `InPortal` — sem `View`, sem
  `Intent`. `bot.SeekPortal` também deixou de usar `seekAndSeparate`: a
  separação existe para o grupo não morrer de um golpe em área, e o portal só
  materializa com a fase já limpa — ali não há golpe em área para evitar, e a
  separação era exatamente o que empurrava os bots para fora do mesmo pequeno
  retângulo. Detalhe completo, incluindo o lado humano (`InPortal`,
  `ProcessInput`, cancelamento por ESC/SAIR), em `doc/network.md` e
  `doc/tilemap.md`.

### Por classe

**Paladina — segurar a linha.** Mede a frente (o grupo de monstros mais próximo
dos aliados frágeis) e se coloca *entre* eles. Como o monstro persegue o jogador
mais **perto** (`FindNearestPlayer`), segurar o aggro aqui é geometria, não uma
estatística de ameaça: basta ser a mais próxima. Escudo quando três ou mais
monstros estão em alcance de corpo, ou quando a própria vida cai abaixo de
metade. Espada sempre que houver alguém no arco. Avatar Divino quando o grupo
está caindo — dois aliados abaixo de 30%, ou um morto com horda em campo.

**Sacerdotisa — manter o grupo de pé primeiro, ferir depois.** Segunda linha,
atrás da Paladina (`backLine = 420`, dentro do alcance de ~760px do tiro —
`boltRange`, espelhando `HolyAttackSpeed * Lifetime`), com fuga imediata de
qualquer coisa mais perto que `panicLine = 260`. A prioridade da mira, a cada
quadro:

1. Existe aliado vivo abaixo de 100% dentro do alcance do tiro? Mira **na reta
   que atravessa o mais ferido** (o tiro cura quem atravessa —
   `Host.checkHolyProjectileHeals` — e **não cura o próprio lançador**, nem é
   consumido pelo aliado). Se essa mesma reta também encontrar um monstro
   depois do aliado, prefere esse ponto — dois efeitos por tiro; senão, atira
   através do aliado mesmo assim, porque curar 12 já vale o tiro.
2. Algum monstro está NO CAMINHO até esse aliado? Então o tiro seria
   consumido antes de curar: não atira nessa reta — ataca o bloqueador, que é
   a mesma decisão de defesa.
3. Ninguém para curar ao alcance (ou todo mundo cheio)? Mira o monstro mais
   ameaçador (`mostThreateningFoe`), com previsão de movimento.

Quando não há monstro nenhum a `calmRadius = 900` dela — a paz LOCAL, não o
contador de horda — ela usa a folga para encher a vida do grupo: se aproxima
de quem está ferido e continua atirando até todos estarem cheios; se a ferida
for ela mesma e o Santuário estiver pronto, lança nos próprios pés (é a única
cura que ela tem); com todo mundo cheio, apenas segue o grupo sem atirar no
vazio.

Santuário no aglomerado de aliados feridos — CONTANDO ela mesma na soma — e só
quando ela já está perto o bastante do aglomerado para a área realmente cair
em cima de alguém (a área nasce nos próprios pés, deslocada). Área Angélica
quando há morto para reerguer, ou três aliados abaixo de 35%.

**Arqueiro — dano de longe, sem parar.** Mantém a distância máxima útil e kita.
Alvo: o que ameaça o aliado mais frágil; empatados, o mais ferido — rematar vale
mais que espalhar dano. Saraivada quando três ou mais inimigos couberem no
leque. Flechas Celestiais contra chefe (`target.IsBoss`) na seleção normal de
alvo, e contra sentinela por uma regra à parte, acima de tudo o mais: suprema
pronta **e** sentinela viva vira a decisão do quadro inteiro
(`arqueiroBrain.huntSentry`) — aproxima até a distância útil da própria
suprema (`skill.CelestialRange` menos uma margem, para o tiro não expirar no
caminho) e então mira o `HitCentre`, preferindo uma segunda sentinela alinhada
atrás da primeira (uma ativação, duas torres). Nenhuma outra classe, nem o
próprio Arqueiro fora dessa regra, pode gastar cadência numa sentinela:
`IsSentry` sai de toda seleção de alvo comum (`nearestFoe`,
`mostThreateningFoe`, `clusterCentre`, `countFoesWithin`, `foeBlocksLine`,
`foeBeyondAlly`, `anyFoeWithin`) — o host já recusa esse dano de qualquer jeito
(`checkProjectileCollisions`/`checkEnemyPlayerCollisions`), então acertar seria
gastar recarga em nada. Ver `doc/plan_avanco_bots_e_gargula.md` §B4.

**Mago — dano em área.** Meia distância. A Bola de Fogo vai no ponto que cobre
mais monstros dentro do raio de explosão (uma varredura sobre os inimigos, não
uma grade), preferindo o aglomerado mais perto dos aliados. **Fogo não queima
aliado** (`skill/fire_ground_damage.go`: *"Fire hurts MONSTERS only"*) — o Mago
larga a zona em cima do corpo a corpo por desenho, então a regra é *não empilhar
zona nova sobre zona ainda acesa*, e não desviar de aliado. Chuva de Meteoros
quando a horda em campo passa do limiar.

**Necromante — controle.** O cemitério é uma faixa de 520×320 lançada à frente
do conjurador, que dura 6 s e reduz a velocidade a 45% (`skill/graveyard.go`).
Ele vai no **caminho** entre a horda e o grupo, não em cima da horda: lentidão
vale pelo tempo que o monstro passa dentro dela. O tiro básico com roubo de vida
no alvo mais próximo é o que o mantém de pé. Legião Espectral quando ele está
cercado, ou quando a fase entrega massa.

---

## 6. Supremas e a narrativa

**Nenhuma tabela nova.** O bot pede a suprema por `HandleSkillMessage`, e
`skillUnlocked` já responde com `UltimateUnlockedFor(char)` — a campanha
(`UltimatesGrantedOn`, derivada de `campaignMaps` + `lastStandHeroes`) mais o
que a corrida concedeu. O efeito é exatamente o pedido: o bot do Necromante só
tem a Legião a partir do mapa 3, o da Sacerdotisa a partir do 4, o do Arqueiro a
partir do 5, o do Mago a partir do 6 e o da Paladina a partir do 7.

Para não pedir o que será negado (log e mensagem à toa), a `View` do bot carrega
`UltimateReady bool`, preenchido pelo host com a mesma pergunta — o pacote `bot`
não pode importar `network` (§4).

**Duas armadilhas para deixar escritas:**

- F2 (modo de teste) é por jogador, e um bot nunca entra em `testPlayers`. O F2
  do host **não** destrava a suprema dos bots. Está certo assim, mas parece
  defeito na hora de testar — para ver a suprema de um bot, chegue ao mapa em
  que ela é liberada (F8 / Shift+F8, `game/stage_skip.go`).
- Paladina e Mago só ganham a suprema nos dois últimos mapas da campanha. Um
  critério de aceite que peça "ver o Avatar Divino do bot no mapa 1" é
  impossível por desenho.

---

## 7. O último suspiro com bot

`reviveHero(character)` já varre `h.players` e reergue o primeiro daquela
classe. Com o bot em `h.players`, isso passa a ser satisfeito sozinho: `saved`
deixa de ser vazio, a suprema é dele, `GrantUltimateForRun` e
`BroadcastUltimateGrant` acontecem, e o NPC não é convocado. É a mudança de
comportamento mais barata do plano — uma consequência, não um código.

Três coisas para acertar:

1. **Preferir humano a bot** quando os dois jogam a classe (§3, classe
   duplicada). Hoje `reviveHero` pega o primeiro da iteração de um mapa, que é
   aleatória. Ordem: humano presente > humano ausente > bot.
2. **O bot reerguido precisa lançar.** No caminho comum, `ResolveLastStand` não
   lança — espera o jogador. O bot lança porque a suprema acabou de ser liberada
   e o gatilho ("o grupo está caindo") está satisfeito; enquanto a concessão for
   recente (2 s), o relógio de reação do bot cai para zero, ou a cena fica
   segurando um silêncio de segundos.
3. **`summonHeroNPC` continua existindo**, para o mapa em que a classe não tiver
   representante nenhum. Não deve acontecer — mas é o que impede a cena de sumir
   em silêncio se acontecer. Os mapas 4 e 6 têm caminho próprio em
   `ResolveLastStand` (julgamento das sentinelas e dos canhões): conferir os
   dois com bot no papel, porque eles escolhem `owner` entre jogador e NPC.

---

## 8. O que este plano não muda

Cliente, protocolo, desenho, interpolação, descoberta, menu e seleção de
personagem. Se a implementação precisar mexer em qualquer um deles, o desenho
saiu do trilho e vale voltar ao §2.

---

## 9. Riscos, e a regra que cobre cada um

1. **Bot preso no cenário segurando a porta do portal e o Game Over.** Detector
   de emperrado (menos de `stuckDistance` em `stuckWindow` com destino ativo) →
   desvio lateral, aproveitando que o resolvedor já desliza pelas paredes; em
   último caso, o host recoloca o bot no centro do grupo. **Nunca** fazer o
   portal ignorar o bot: isso desfaria a decisão de que ele conta em tudo.
2. **Custo por quadro.** Cinco bots varrendo oitenta monstros é barato, mas não
   a cada quadro: alvo e aglomerado saem dos relógios de decisão, e o resto do
   quadro é só andar.
3. **Log no caminho quente.** `HandleAttack` loga **todo** ataque
   (`host.go`, `"[Host] Player %s fired ..."`), e `HandleSkillMessage` loga toda
   recusa. Com quatro bots atacando 1,2–2,5 vezes por segundo isso vira uma
   enxurrada — este repositório já pagou esse preço duas vezes
   (`doc/performance.md`). Rebaixar esses três logs faz parte da entrega.
4. **Concorrência.** `tickBots` roda na goroutine da simulação. Monte a `View`
   sob `playersMutex.RLock` e **solte antes** de decidir e atuar: `HandleAttack`
   toma `playersMutex` por conta própria e, dentro dele, `cdMutex`. Chamar
   qualquer um dos dois com `playersMutex` na mão é `deadlock` imediato. Ordem
   de aquisição em todo o host: `playersMutex` → `cdMutex`; nunca o contrário.
5. **Empilhamento.** Ver a separação no §5. Sem ela, um único golpe em área
   apaga o grupo inteiro.
6. **Reset, viagem e reconexão.** Bot é jogador, então já está coberto — desde
   que `ReconcileBots` seja chamada dos quatro pontos do §3 e que a IA descarte
   destinos do mapa anterior (o `botRuntime` inteiro é zerado na troca de mapa).
7. **Dificuldade.** O jogo passa a ter sempre cinco em campo — a régua para a
   qual as hordas foram escritas. Solo deixa de ser inviável, e a percepção de
   dificuldade da campanha inteira muda. **Jogar a campanha toda e medir antes
   de afinar número nenhum.**

---

## 10. Ordem de implementação

| Fase | Entrega | Critério de aceite |
|---|---|---|
| 0 | **Poda de `RemotePlayers`** (§4). | Teste: jogador removido de `h.players` some de `GetAllPlayers()` no host. |
| 1 | **A vaga**, sem inteligência: o bot nasce, é desenhado, apanha, morre e ressuscita. | Host sozinho como Mago vê quatro personagens **das quatro outras classes** (não quatro Magos) parados no spawn; matar todos dá Game Over; F5 repõe. |
| 2 | **Posse e herança.** | Cliente entra com a classe do bot e assume o corpo com a vida que ele tinha; classe duplicada cria personagem novo; ausência de 5 min devolve a vaga ao bot; nenhum efeito órfão fica na tela do cliente. |
| 3 | **Movimento**: seguir o grupo, colisão, separação, portal. | Um humano só atravessa do mapa 1 ao 3 com o grupo inteiro, sem F8. |
| 4 | **Combate por classe** — ataque básico e habilidade primária, uma classe de cada vez: Paladina, Sacerdotisa, Arqueiro, Mago, Necromante. | Cada classe sozinha com o host, contra a horda do mapa 1, faz o que o §5 descreve para o básico e o Q. Supremas **não** entram aqui. |
| 5 | **Supremas e último suspiro.** | Bot do Necromante não lança nada no mapa 2 e lança no 3 (F8 para chegar); a cena do mapa 2 é resolvida pelo bot, sem NPC; mapas 4 e 6 conferidos. |
| 6 | **Afinação.** | Campanha inteira jogada e medida. Só aqui os números de `tuning.go` mudam. |

---

## 11. Testes

- **`bot`**: as decisões são funções puras e testáveis sem janela — escolha de
  alvo, ponto da Bola de Fogo, a reta de cura da Sacerdotisa, o gatilho de cada
  suprema. O mesmo formato de `wave_runs_test.go` e `climax_window_test.go`.
- **`network`**: `ReconcileBots` (uma vaga por classe; humano ausente segura a
  vaga; duplicado não remove bot; roster transmitido); posse (recarga migra,
  efeito no ar some, sinal de fim transmitido); `reviveHero` (humano preferido
  ao bot); poda de `RemotePlayers`.
- **`game`**: a porta do portal abre com bots no grupo; o Game Over só chega
  quando todos caem.
- Nenhum teste pode abrir janela raylib — siga o que os testes existentes fazem.

---

## 12. Em aberto

- **Rótulo na tela.** Esta entrega vai **sem** rótulo. Se for pedido depois, é
  campo de identidade (`Bot bool` em `PlayerState`, `omitempty`), anunciado pelo
  roster e nunca no tique — e aí deixa de ser verdade que o cliente não muda.
- **Grupo de seis** quando a classe é duplicada: aceito agora, revisitar se a
  campanha ficar fácil.
- **O bot fala?** Os diálogos assumem personagens e o narrador é o host — nada
  muda hoje, mas fica registrado para quando um roteiro citar quem está em campo.

---

## 13. Contrato de execução

Esta seção existe para que quem implementar não precise inventar nada.

### 13.1 Fluxo obrigatório do repositório

1. Ler `AGENTS.md` e `doc/coding_patterns.md` antes de escrever.
2. `git status --short` antes e depois.
3. Um arquivo, uma responsabilidade; dividir antes de passar de ~150 linhas.
4. Nada de lógica de jogo em `internal/game/loop.go`.
5. `assets.Path()` para todo carregamento (não deve aparecer aqui, mas vale).
6. Ao final de cada fase: `gofmt -l .` (vazio), `go vet ./...`,
   `go build ./...`, `go test ./...`, e rodar o desktop (`doc/running_desktop.md`).
7. Registrar o pacote novo `internal/bot` na tabela de `doc/coding_patterns.md`,
   atualizar `doc/network.md` (a seção de jogadores) e **acrescentar uma linha a
   `doc/changelog.md`** por fase entregue.

### 13.2 Assinaturas

```go
// internal/bot
type Ally struct { ID string; Char entity.CharacterType; Pos rl.Vector2; Health, MaxHealth float32; IsDead, IsBot bool }
type Foe   struct { ID string; Pos, Vel rl.Vector2; Health, MaxHealth float32; AttackRange, Radius float32; IsBoss bool }

type View struct {
    Self          Ally
    Allies        []Ally   // sem o próprio bot
    Foes          []Foe
    Bounds        world.Bounds
    PartyCentre   rl.Vector2
    Portal        rl.Vector2 // destino quando PortalActive
    PortalActive  bool
    EnemiesLeft   int        // horda em campo + por nascer
    PrimaryReady  bool       // recarga da habilidade Q
    UltimateReady bool       // recarga E liberação narrativa, já resolvidas pelo host
    RescueRecent  bool       // o resgate acabou de conceder a suprema (§7.2)
    Dt            float32
}

type Cast struct { SkillID string; Aim rl.Vector2 }
type Intent struct { Move rl.Vector2; Attack *rl.Vector2; Skill *Cast }

type Brain interface{ Think(View) Intent }
func For(char entity.CharacterType) Brain

// internal/network
func (h *Host) ReconcileBots()
func (h *Host) SetPartyPortals(rects []rl.Rectangle, active bool)
func (h *Host) tickBots(dt float32)          // chamada de UpdateSimulation
func (h *Host) takeOverBot(char entity.CharacterType, newID string) bool

// internal/skill
func (m *Manager) ClearOwner(ownerID string) (endSignals []string)

// internal/entity
func WalkRowFor(dir rl.Vector2) int
func StepWalkAnimation(def CharacterDef, frame int, timer float32, moving, sprinting bool, dt float32) (nextFrame int, nextTimer float32)
```

### 13.3 Estado por bot (host)

`h.bots map[string]*botRuntime`, protegido pelo mesmo `playersMutex` que
`h.players` — é estado da mesma vaga.

```go
type botRuntime struct {
    animTimer   float32
    frame       int
    lastRow     int
    targetID    string   // inimigo escolhido
    decideIn    float32  // relógio de reavaliação
    reactIn     float32  // relógio de reação
    lastPos     rl.Vector2
    stuckFor    float32
}
```

`PlayerState` **não** ganha campos para isso: ele é o que vai no fio.

### 13.4 Onde `tickBots` entra em `UpdateSimulation`

Depois de `h.updateWaves(dt)` e **antes** de `h.EntityManager.UpdateAll(...)`:
o bot decide com o mundo do quadro anterior — como um jogador, cujo input também
chegou antes da simulação — e o movimento dele já vale para o alvo que o monstro
escolhe neste quadro.

### 13.5 Valores iniciais de `tuning.go`

São chutes informados, para serem afinados na fase 6. Não invente outros; mude
estes.

| Constante | Valor | Por quê |
|---|---|---|
| `decideEvery` | 0,25 s | Abaixo disso o bot vibra entre alvos empatados. |
| `reactDelay` | 0,25 s | Faixa humana boa; zero lê como trapaça. |
| `allySeparation` | 90 px | Pouco mais que a largura visível do personagem. |
| `followRadius` | 700 px | Menos que a tela: o bot some antes de ser abandonado. |
| `stuckDistance` / `stuckWindow` | 40 px / 1,5 s | Um passo real em um segundo e meio. |
| Paladina: `frontRing` / `shieldFoes` | 140 px / 3 | Alcance da espada, e "cercada". |
| Sacerdotisa: `backLine` / `sanctuaryAllies` | 300 px / 2 abaixo de 70% | Fora do corpo a corpo, dentro do alcance do tiro. |
| Arqueiro: `keepRange` / `retreatUnder` | 600 px / 320 px | Kite que ainda acerta. |
| Mago: `keepRange` / `clusterMin` | 450 px / 3 | A explosão precisa valer a recarga. |
| Necromante: `graveyardMin` | 4 inimigos vindo | A faixa é 520×320: menos que isso desperdiça 12 s de recarga. |

---

## 14. Armadilhas verificadas no código

Cada item foi conferido no repositório em 20/08/2026. São os pontos em que uma
leitura razoável do código levaria ao erro.

1. **O host não sabe o que é um portal.** Portais vivem em `game.World`, e
   `game` importa `network` (nunca o contrário). Por isso `SetPartyPortals`: o
   `game`, que já conta a festa a cada quadro em `UpdatePortal`, entrega ao host
   os retângulos (não só um ponto — desde a espera no portal, o host precisa da
   caixa inteira para saber quem está DENTRO, não só para onde apontar um bot) e
   um booleano. Não mova lógica de portal para a rede.
2. **`RemotePlayers` não é podado** — ver fase 0 (§4). É pré-requisito, não
   melhoria.
3. **No host, o desenho dos outros jogadores anda a 20 Hz.**
   `InterpolatedPlayers` devolve o estado direto quando o papel não é cliente, e
   `RemotePlayers` só é escrito no tique de publicação (`SnapshotHz`). Um
   jogador humano remoto já sofre disso; com quatro bots na tela do host — o
   caso mais comum — vira o defeito visível da fase 1. `tickBots` publica cada
   bot em `RemotePlayers` **a cada quadro** (escrita local, sem rede). E, como
   `UpdatePlayerState` avisa: **não transmitir dali**, ou a taxa de snapshot
   deixa de existir.
4. **Identidade só viaja no roster** (`slimPlayer`, `wire.go`) — §3.
5. **As coleções de efeito têm duas chaves** — §4, `ClearOwner`.
6. **Não existe `tickLegions`.** A manutenção da Legião e do Cemitério é
   `handleNecroTick` (`host_necro.go`); a Área Angélica é um altar **fixo**, não
   ancorada (`host_ultimate.go`). Quem limpar efeitos por dono precisa emitir o
   sinal de fim que essas funções emitiriam.
7. **`HandleAttack` toma `playersMutex`; `HandleSkillMessage` também, via
   `PlayerState()`** — §9.4.
8. **Fogo não fere aliado** (`fire_ground_damage.go`) — §5, Mago.
9. **A cura da Sacerdotisa não vale para ela mesma**
   (`checkHolyProjectileHeals` pula `playerID == proj.OwnerID`) — §5.
10. **Vida de jogador é 100/100 no código, para toda classe**
    (`entity.NewPlayer`, `StartHost`, `handleJoin`). O bot nasce igual; se algum
    dia houver vida por classe, é uma mudança de outro plano.
11. **`internal/system` é código morto** — §4.
12. **Uma trava de posição não é colisão, e por isso não vale de graça para
    bot.** A trava de mão única da arena do mapa 5 (`game/arena_gate.go`, ver
    `doc/tilemap.md` "Arena de mão única") é uma correção de posição aplicada
    a UM `*entity.Player` — sempre o jogador local de quem chama. Um humano
    obedece porque o próprio cliente dele roda a checagem todo quadro; um bot
    é um corpo que só o host move, e nenhum cliente roda checagem nenhuma por
    ele. Toda regra de movimento pensada como "correção aplicada ao jogador
    local" precisa do mesmo tratamento que este conserto deu: um canal
    `network.Set*` (padrão de `SetPartyPortals`) que publique o estado por
    quadro, e a correção reaplicada no host, por bot — nunca ao humano, cuja
    posição já é autoridade do cliente dele.
