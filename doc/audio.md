# Áudio: trilha de fase e efeitos das magias

Tudo mora em `internal/sound`. Um único dispositivo de áudio por processo:
`sound.Init()` e `sound.Shutdown()` são chamados em `game/loop.go:Run`, ao
redor do laço de sessões — nunca por sessão.

## Trilha (music.go)

Uma faixa por mapa, em `mapTracks` (chave = nome-base do arquivo do mapa),
tocando em laço. `sound.PlayForMap(path)` troca a trilha e é chamada em dois
pontos: `playSession` no mapa inicial e `travelTo` (portal e F8). É no-op se o
mapa pedido já é o que está tocando. Volume: ver "Barramentos de volume"
abaixo — deixou de ser a constante `Volume` (0.5) fixa.

### O sub-buffer, e por que ele não é 33 ms

`sound.Update` alimenta o stream **uma vez por quadro do jogo**. O padrão do
raylib para o sub-buffer de um `AudioStream` é `sampleRate/30`, ou seja ~33 ms:
com ele, a música só não engasga enquanto o jogo mantiver **todo** quadro
abaixo de 33 ms, e um único quadro de 40 ms já produz uma falha audível.

Foi exatamente o que se ouviu nos mapas 2 e 3 (o defeito era de render, não de
áudio — ver `doc/performance.md`, seção 1.5). `musicBufferFrames = 4096` sobe a
folga para ~93 ms a 44,1 kHz, e é aplicado com
`rl.SetAudioStreamBufferSizeDefault` **antes** de `InitAudioDevice`, porque o
valor é lido na criação de cada stream — mudá-lo depois não alcança a trilha já
carregada.

O que isto **não** é: conserto de fps. É a remoção de um acoplamento — queda de
quadro deve custar fluidez de imagem, não áudio picotado. Os efeitos das magias
não são stream (`rl.Sound` carrega o arquivo inteiro na memória) e não são
afetados.

## Barramentos de volume (bus.go) — Configurações

Desde a tela de Configurações (`doc/settings.md`), `Volume` (música) e
`SFXVolume` (efeitos) **deixaram de ser constantes**. Viraram quatro
barramentos atômicos (`sync/atomic`, porque `enqueue` roda de qualquer
goroutine — o cliente replica magias na goroutine que lê a rede):

| Barramento | Setter | Governa |
|---|---|---|
| Geral (master) | `sound.SetMasterVolume` | multiplica os outros três |
| Música | `sound.SetMusicBusVolume` | `music.go`, aplicado NA HORA (`applyMusicVolume`) |
| Habilidades | `sound.SetSkillVolume` | os doze efeitos de magia (`sfxDef.bus = busSkill`) |
| Ataque básico | `sound.SetAttackVolume` | os cinco `*_attack.mp3` (`busAttack`) |

Cada barramento tem um mudo próprio (`SetMute{Master,Music,Skill,Attack}`)
que silencia SEM perder o número do slider. Há também
`SetFocusSuppressed`, usado só por `internal/game/audio_focus.go` para
"silenciar em segundo plano" — independente do mudo manual do jogador.

**Como cada disparo usa isto.** `enqueue` (sfx.go) multiplica a atenuação de
distância pelo barramento da CATEGORIA do efeito (`busVolume(def.bus)`) — a
categoria é uma coluna a mais em `sfxDef`, não uma segunda tabela. Um efeito
já tocando não muda de volume no meio (o próximo disparo já sai no valor
novo); a exceção é a música, contínua, que responde na hora porque é o que
o jogador está ouvindo enquanto arrasta o slider.

**`sound.PlayUI(id SFX)`** toca um efeito SEM espacialização, no volume
cheio do barramento — nem atenuação por distância, nem exige
`SetListener`. É o que a prévia sonora do slider de volume usa
(`ui/settings_section_audio.go`): na tela de Configurações não existe
jogador local, então `PlayAt` silenciaria tudo por falta de ouvinte.

## Efeitos das magias (sfx.go, sfx_bank.go)

**A regra:** um efeito só é ouvido por quem está perto de onde ele nasceu.

- O ouvinte é o jogador local: `sound.SetListener(p.Position)`, uma vez por
  quadro em `game/loop.go`.
- `attenuation` dá volume cheio até `SFXFullRadius` (400) e cai
  quadraticamente até o silêncio em `SFXHearRadius` (1600). São **unidades de
  mundo**; com o zoom fixo em 1.0, 1600 é um pouco além da borda da tela em
  1080p.
- Passou do raio, o disparo nem entra na fila.

**Onde o som é disparado:** no `internal/skill`, no ponto em que o efeito
NASCE — `SpawnFireball`, `AddExplosion`, `spawnFanLocked`, `Meteor.Advance` no
quadro do impacto, e assim por diante. Aquele código roda igual no host e em
todo cliente (o cliente replica a magia pelo evento que já existia), então
cada máquina resolve sozinha o que o dono dela ouve, **sem uma única mensagem
de rede a mais**. Não há caminho separado para "magia minha" e "magia dos
outros".

**A fila.** `PlayAt` é chamada de qualquer goroutine (o cliente replica magias
na goroutine que lê a rede). Ela só enfileira; toda chamada de áudio do raylib
acontece em `updateSFX`, dentro de `sound.Update()`, na thread do laço.

**Vozes.** Um `rl.Sound` toca um disparo por vez — mandar tocar de novo
reinicia o que estava tocando. Cada efeito carrega `sfxDef.voices` cópias do
arquivo e os disparos giram entre elas. `sfxDef.minGap` é o intervalo mínimo
entre dois disparos do mesmo efeito: a Chuva de Meteoros derruba ~67 pedras por
segundo e sem isso o impacto viraria ruído branco.

**Prazo e desvanecimento.** `PlayAtFor(id, pos, life)` é para o efeito cujo som
tem de acabar junto com a magia: toca normalmente e desvanece na janela final
(`sfxFadeOut`, no máximo 40% de `life`). O prazo vem da constante da própria
magia (`SanctuaryDuration`, `FireGroundDuration`, `GraveyardDuration`,
`AngelicDuration`), que é onde ele já estava definido — por isso `sound` não
importa `skill`, e sim o contrário.

| Personagem | Q | R |
|---|---|---|
| Sacerdotisa | `Sacer_Q.mp3` (prazo: aura) | `Sacer_R.mp3` (prazo: área) |
| Mago | `Fireball-lauch.mp3` ao lançar, `fireball-impact.mp3` no impacto, `flames.mp3` no chão em chamas (prazo: zona) | `meteor.mp3` a cada impacto |
| Necromante | `Necro_Q.mp3` (prazo: cemitério) | `Necro_R.mp3` |
| Arqueiro | `Arch_Q.mp3`, uma vez por leva (`ArrowVolleyWaves`) | `Arch_R.mp3` |
| Paladina | `Pala_Q.mp3` | `Pala_R.mp3` |

## Ataque básico

Um som por personagem, declarado em `CharacterDef.AttackSound`
(`entity/character.go`) ao lado de `AttackSpeed` — registrar um personagem
novo continua bastando. `entity.PlayAttackSound(ct, pos)` é a única fonte
dessa resposta, e corta o som em `AttackInterval()` com desvanecimento: todos
os cinco arquivos são mais longos que a cadência (o do Necromante tem 8 s para
uma cadência de 0,77 s), então sem prazo os golpes se empilhariam.

| | Arquivo | Cadência |
|---|---|---|
| Mago | `Mago_attack.mp3` | 1,2/s |
| Sacerdotisa | `Sacer_attack.mp3` | 1,4/s |
| Paladina | `Pala_attack.mp3` | 1,5/s |
| Necromante | `Necro_attack.mp3` | 1,3/s |
| Arqueiro | `Arch_attack.mp3` | 2,5/s |

**Três pontos de disparo, e não um só, porque a replicação é diferente da das
magias:**

- **Paladina** — `skill.NewSwordSweep`. O golpe dela tem evento próprio
  (`MsgMelee`), então host e cliente criam a mesma varredura e um gancho
  atende os dois, exatamente como nas magias.
- **Os outros quatro, no host** — `Host.HandleAttack`, no ramo do projétil.
- **Os outros quatro, no cliente** — `network/client_attack_sound.go`. O
  ataque de projétil **não tem evento de rede**: ele aparece no instantâneo
  (`ProjectileUpdatePayload`) e some dele. Então "alguém atacou" é deduzido —
  um id de projétil que não estava no instantâneo anterior é um disparo novo.
  Continua custando zero de rede, e criar uma mensagem por golpe seria a
  opção mais cara possível, porque ataque básico é o que mais acontece na
  partida. O primeiro instantâneo da sessão (início, troca de mapa,
  reconexão) não soa: ali tudo é novidade porque o cliente estava fora.

## Adicionar um efeito novo

1. Ponha o `.mp3` em `assets/sounds/`.
2. Declare o id e a linha em `sfxDefs` (`sound/sfx.go`).
3. Chame `sound.PlayAt` / `PlayAtFor` no ponto do `internal/skill` onde o
   efeito nasce — o mesmo que host e cliente executam. Nada de evento novo de
   rede, e nada em `internal/network`.
