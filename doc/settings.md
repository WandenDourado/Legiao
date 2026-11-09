# Configurações

Tela de opções locais: áudio, vídeo/desempenho, controles, interface e
jogador. Acessível pelo botão CONFIGURAÇÕES no menu inicial e, de dentro de
uma partida, por ESC (desktop) ou pela engrenagem do HUD, em toda plataforma
— encostada à esquerda do contador "Horda N/M" (ver "A engrenagem da
partida", abaixo). A partida **não pausa** enquanto a tela está
aberta — ela é desenhada por cima e o jogo continua simulando, exatamente
como a sobreposição de reconexão.

## Restrição não-negociável

**Configuração é LOCAL de cada máquina e nunca entra na rede.** Nenhum campo
de `internal/settings` é lido por `internal/network`, nenhum é enviado numa
mensagem, e o pacote `settings` não importa `network`. Isso guiou duas
decisões de corte abaixo (nome do jogador, mostrar nomes).

## Onde cada coisa mora

| Camada | Pacote/arquivo | Responsabilidade |
|---|---|---|
| Modelo + persistência | `internal/settings` (`settings.go`, `defaults.go`, `store.go`, `store_desktop.go`, `store_android.go`) | struct `Settings`, `Default()`, clamp por campo, `Get`/`Set`/`Update` (RWMutex, leitura barata), `Load`/`Save` em JSON |
| Barramentos de áudio | `internal/sound` (`bus.go`, `sfx_preview.go`) | 4 barramentos atômicos (master/música/habilidades/ataque) + mudo por barramento + "silenciar em segundo plano"; `PlayUI` para a prévia sem espacialização |
| Controle de volume | `internal/ui` (`slider.go` matemática pura, `slider_input.go` toque/mouse/roda) | posição↔valor, clamp, degrau de 5%, arraste com toque real + mouse |
| A tela em si | `internal/ui` (`settings_screen.go`, `settings_layout.go`, `settings_widgets.go`, `settings_textfield.go`, `settings_section_*.go`, `settings_apply*.go`, `settings_access_button.go`, `settings_scale.go`) | abas, linhas, aplicar-na-hora, entrar/sair |
| Geometria de toque | `internal/input/control_prefs.go` (+ edições em `touch.go`) | escala do botão, raio do joystick, modo canhoto — multiplicadores em cima da geometria existente |
| Acesso de dentro da partida | `internal/game/settings_overlay.go`, `internal/game/audio_focus.go`, `internal/game/desktop_aim.go` | abrir/fechar sem pausar, resolução do conflito com ESC do portal, silenciar em segundo plano, mira do mouse |

## As opções

### Áudio
| Opção | Aplica em | Observação |
|---|---|---|
| Volume geral | `sound.SetMasterVolume` | multiplica os outros três |
| Música | `sound.SetMusicBusVolume` | responde NA HORA (é contínua) — `PlayForMap` também lê o barramento |
| Habilidades | `sound.SetSkillVolume` | as doze magias do catálogo (`sfxDef.bus = busSkill`) |
| Ataque básico | `sound.SetAttackVolume` | os cinco `*_attack.mp3` (`busAttack`) |
| Mudo por barramento (4×) | `sound.SetMute{Master,Music,Skill,Attack}` | preserva o número do slider — desmutar volta exatamente onde estava |
| Silenciar em segundo plano | `game/audio_focus.go` → `sound.SetFocusSuppressed` | ver "Decisões", item Foco |

**Prévia sonora**: ao SOLTAR o slider de Habilidades toca `SFXFireLaunch`, o
de Ataque toca `SFXAttackMago`, via `sound.PlayUI` (sem espacialização — a
tela de Configurações não tem jogador local, `SetListener` nunca foi
chamado). O de Música não tem prévia própria: a trilha já está tocando e
muda de volume em tempo real, que É a prévia. O de Volume geral também não
toca nada sozinho — não tem timbre próprio.

### Vídeo e desempenho
| Opção | Aplica em | Observação |
|---|---|---|
| Teto de quadros (30/60/sem teto) | `rl.SetTargetFPS`, na hora | também persistido, aplicado de novo no próximo início |
| VSync | `Config.VSync` → hint de contexto GL | **só vale na próxima abertura do jogo** — a opção já avisa isso na tela |
| Tela cheia (desktop) | `rl.ToggleFullscreen` | build Android trata como no-op (a tela já preenche por `Config.FullScreen`, que ali significa "layout Android", não "modo janela") |
| Painel de desempenho (F3) | `tilemap.ToggleDebug()` | agora alcançável no celular, não só por tecla |
| Qualidade de efeitos das magias | — | **ainda fora**, ver "O que ficou de fora" |

### Gráficos (aba própria desde 28/09/2026)
| Opção | Aplica em | Observação |
|---|---|---|
| Qualidade (Baixa/Média/Alta) | `settings.ApplyPreset` (`settings/quality.go`) | nome para uma combinação: Alta = 60 fps, cinzas, flores; Média = 60 fps, cinzas, sem flores, sugere 900p; Baixa = 30 fps, sem cinzas, sem flores, sugere 720p. Mexer num campo à mão mostra "personalizada". A resolução NÃO entra na comparação — é eixo próprio |
| Configurar sozinho (TESTAR) | `ui/perf_bench*.go` | teste de ~2–8 s: cena sintética ~30% mais pesada que uma luta cheia, desenhada em cada resolução candidata (RenderTexture do tamanho dela). **Mede o custo da cena entre duas cercas de GPU** (`perf_bench_gpu.go`: RenderTexture 1×1 lida de volta), mediana por resolução — não o tempo do quadro, que carregava vsync, simulação e a própria tela (desde 29/09/2026). Orçamento: cena ≤ 12 ms → 60 fps (Alta se nativa, Média se menor); senão a melhor ≤ 24 ms a 30 fps (Baixa). O resultado mostra o número (`ALTA: NATIVA a 60 fps (2.4 ms)`) e cada resolução vai ao logcat com a marca `[Graficos]`. Grava na hora. O mundo não é desenhado enquanto roda; sair/VOLTAR cancela; as abas travam. Não mede CPU: aparelho com "simulação" alta no F3 pede 30 fps na mão |
| Resolução (celular) | `game/render_scale*.go`, `settings.WindowSize`, `ui/render_restart.go` | Nativa/900p/720p/540p. A janela nasce menor e o scaler de hardware do Android amplia (ramo "Upscaling" do raylib → `ANativeWindow_setBuffersGeometry`). **Só vale quando a janela nasce** — sair pelo Home e voltar NÃO conta, o processo é o mesmo. A dica da linha mostra o tamanho valendo agora (`agora: 1560x720, ampliada pela tela`); quando difere do escolhido aparece **REINICIAR AGORA** no canto do painel (`internal/apprestart`: `Intent.makeRestartActivityTask` via JNI e fim do processo). Dentro de uma partida o botão vira **REINICIAR E VOLTAR** e o jogo reabre direto nela (`internal/resume` + `ui/resume_session.go`): o cliente volta exatamente onde estava (mesmo `PlayerID`, slot guardado pelo host); o host — e o solo — volta no começo da fase atual, mesmo personagem e cor. Host com outras pessoas vê o aviso de que a partida cai para todos; eles reconectam sozinhos — na rede local ao mesmo IP, online ao mesmo CÓDIGO de sala, que o host pede de volta ao relay com o segredo dela (`relay.HostAs`). O zoom da câmera é reescalado para o jogador ver o MESMO pedaço de mapa, e a interface também: todo pixel fixo dela é pixel da tela nativa (`ui.Px`, `ui/text.go`), então trocar a resolução muda a nitidez, nunca o tamanho da HUD. No PC não existe: o jogo já desenha na resolução do monitor, e acima disso só gasta |
| Monstros viram cinzas | `entity.SetDeathFXEnabled` | desintegração (`doc/combat_rules.md`, "Morte dos monstros"); desligar apaga na hora o que estiver em curso |
| Flores e galhos no chão | `tilemap.SetGroundDetailVisible` | a camada de TILES `ground_detail`; os sigilos da arena (peças de manifesto) ficam. Medido: ~10% da tela, ganho pequeno |

A aba Gráficos **grava a cada toque** (`applyGraphics`), ao contrário das
outras, que gravam só no VOLTAR: fechar o app direto da tela era o jeito
natural de "reabrir" e perdia a resolução escolhida.

`DisplayWidth/DisplayHeight` guardam o tamanho nativo da tela, anotado a cada
abertura em resolução nativa (`noteDisplaySize`, grava na hora): o Android
não informa a proporção da tela antes da janela existir, e a janela
escalada precisa dela para não ganhar tarja. Primeira abertura depois de
instalar = sempre nativa.

### Controles — mobile (build tag `android`)
| Opção | Aplica em |
|---|---|
| Tamanho dos botões de ação | `input.SetActionButtonScale` (0.8–1.3× sobre `attackRadiusFrac`) |
| Opacidade dos controles | `ui.SetActionOpacity` (multiplica o `dim` de `DrawActionButton`) |
| Modo canhoto | `input.SetLeftHanded` — espelha o centro do cluster E o ângulo da órbita das habilidades, e o centro do joystick de movimento |
| Raio do joystick | `input.SetJoystickRadiusScale` (0.7–1.5× sobre o raio base já existente, `TouchState.SetMaxRadius`) |
| Joystick fixo ou dinâmico | — fora desta rodada |

### Controles — desktop (build tag `!android`)
| Opção | Aplica em |
|---|---|
| Mira segue o mouse continuamente | `game/desktop_aim.go` → `aj.AimDir` todo quadro |
| (padrão) Mira só no clique | `aj.AimDir` só é escrito no instante do clique de ataque |
| Remapear teclas | — fora desta rodada, fase 2 |

### Interface
| Opção | Aplica em |
|---|---|
| Escala do HUD e do texto | `ui.SetHUDScale` — **todo** o texto do HUD, via `ui.TextSize`/`ui.FitTextSize` (`internal/ui/text.go`). Faixa `0.8..settings.HUDScaleMax` (1.8) |
| Mostrar indicador de mira | `game/aim_hint.go` (`drawAimHint`) |
| Mostrar nomes dos jogadores | — fora desta rodada, ver abaixo |

### Jogador
| Opção | Aplica em |
|---|---|
| Nome do jogador | só local, campo de texto (`ui/settings_textfield.go`) — ver "Decisões" |
| Restaurar padrões | `settings.Default()` + `applyAllSettings` reaplicado em todo pacote |

### A engrenagem da partida (28/09/2026)

Ela ficava sozinha no canto superior direito, com ~2% da altura da tela de
diâmetro, e o Gui relatou que era difícil de achar e de acertar. Agora
(`ui/settings_access_button.go`):

- Mora imediatamente à **esquerda** do contador "Horda N/M"
  (`ui.waveCounterLeft`, a mesma conta que desenha o contador, levando em
  conta a linha "restam N" quando ela é mais larga), centrada na linha do
  título. Sem contador na tela (mapa sem horda, fase limpa) ela volta ao
  canto.
- Diâmetro = maior entre 1,35× a fonte do contador (acompanha a escala do
  HUD), 5% da altura da tela e 34 px.
- Alfa 0,45 → 0,7.

### settings.json antigo não desliga opção nova (28/09/2026)

`Load()` passou a decodificar POR CIMA de `Default()`, não de um `Settings`
zerado. Um arquivo gravado antes de um campo existir não o traz, e o zero de
um bool é "desligado" — sem isto, quem já tinha salvo qualquer preferência
abriria o jogo com a desintegração dos monstros desligada.

### A escala do HUD deixou de ser só a barra de vida (20/09/2026)

Os testadores disseram que o texto da HUD e dos diálogos era pequeno demais. A
tela já tinha "Escala do HUD" desde 04/09, mas ela era lida por **um** lugar —
`DrawHealthBar`. O jogador aumentava a escala, a barra crescia e o resto do
texto ficava igual.

O que mudou:

- `internal/ui/text.go` é o ponto único. `TextSize(base)` aplica a escala do
  jogador e o piso (`MinTextSize`, 18); `FitTextSize(size)` aplica **só** o
  piso, para um tamanho que já veio de geometria escalada (o número dentro do
  pip de recarga, o rótulo dentro do botão de ação) — aplicar a escala de novo
  dobraria o fator.
- Fora dessa regra, de propósito: texto em espaço de **mundo**
  (`game/portal_counter.go`), que já cresce e encolhe com o zoom da câmera.
- Os tamanhos de fábrica subiram (contador de horda 22→28, aviso de névoa
  46→52, "Ressuscitando em" 30→38, barra de vida `sw*0.015`→`sw*0.019`).
- O diálogo ganhou faixa por papel em vez de um piso/teto para os três:
  corpo `0.030` da altura com limites 24..44, nome `0.034` com 24..50, dica
  `0.024` com 18..34. O teto antigo (34) era o que impedia o texto de crescer
  num monitor grande; o piso antigo (16) era o que o deixava ilegível numa
  janela 1280x720.
- A caixa de diálogo passou a derivar a altura do texto que promete caber
  (`dialogueBoxHeight`), e o desenho **encolhe** a fonte em vez de cortar a
  fala quando ela ainda não cabe (`fitDialogueBody`). Perder tamanho é
  reversível e visível; perder o fim de uma fala de narrativa, não.
- A faixa do slider foi de `0.8..1.3` para `0.8..settings.HUDScaleMax` (1.8).
  As escalas de **controle** (botão de ação, joystick) continuam nas faixas
  antigas: aquelas mexem em alvo de toque, não em legibilidade.

Continua valendo a restrição não-negociável: nada disto entra na rede.

## Decisões (pontos em aberto que o pedido deixou para o executor)

**Persistência no Android — RESOLVIDA em 28/09/2026.** `rl.HomeDir()` do
raylib-go v0.55.1 devolve `ANativeActivity.internalDataPath` no build
Android (armazenamento privado do app, sem permissão), e pode ser chamado
antes de `InitWindow` — o raylib guarda o `android_app` na primeira linha de
`android_main`, antes de chamar o Go. `store_android.go` grava
`settings.json` ali. Até esta data a configuração no Android valia só a
sessão, o que tornaria os presets de qualidade inúteis justo no aparelho que
precisa deles. **Não compilado para Android neste ambiente** (sem NDK) —
conferir no primeiro build. No desktop, `os.UserConfigDir()/Legiao/settings.json`.

**Ordem de carregamento.** O pedido diz para chamar `settings.Load()` "logo
depois de `sound.Init()`" — mas `sound.Init()` já roda depois de
`rl.InitWindow`, e o VSync só tem efeito como hint **antes** da janela abrir.
Resolvido chamando `settings.Load()` uma vez, ANTES de `InitWindow`, usando o
resultado tanto para o hint de VSync quanto (depois de `sound.Init()`) para
aplicar os barramentos de áudio — um único `Load()`, sem ler o arquivo duas
vezes, cobrindo as duas exigências.

**Nome do jogador não viaja pela rede.** O pedido lista "aparece no grupo e
na descoberta de partidas" como algo bem-vindo, mas a restrição
não-negociável do próprio pedido é que Configurações nunca entra na rede.
As duas frases se contradizem; a restrição, marcada explicitamente como não
negociável, venceu. `settings.PlayerName` é editável e persistido, mas
**não** foi acrescentado a `JoinPayload`/`PlayerState`
(`internal/network`) — fazer isso é uma mudança de protocolo replicado, fora
do escopo desta tela local. Por isso "Mostrar nomes dos jogadores"
(Interface) também ficou de fora: não há nome nenhum vindo da rede para
mostrar. O `PlayerID` (`player_<nanotime>`, `ui/menu.go`) continua
intocado — é o que `host_rejoin.go` usa para reconhecer reconexão, e nunca
deveria virar texto digitado.

**Silenciar em segundo plano.** `rl.IsWindowFocused()` é o único sinal
cross-platform que o `raylib-go` expõe sem uma chamada específica de
plataforma. Não foi possível confirmar neste ambiente se ele reflete "app
minimizado" no Android da mesma forma que reflete "janela sem foco" no
desktop — é o candidato razoável e é o que está implementado
(`game/audio_focus.go`), mas **precisa ser testado em campo** (ver lista de
verificação).

**Teclado do nome no Android.** O campo de texto (`ui/settings_textfield.go`)
lê `rl.GetCharPressed()`/Backspace/Enter — funciona de imediato com um
teclado físico. Ele **não invoca o teclado virtual do Android**: o
`raylib-go` não faz isso sozinho, e integrar um é trabalho de plataforma que
esta rodada corta. Um jogador sem teclado físico no celular não consegue
digitar um nome novo ainda (consegue ver o que já foi digitado no desktop,
já que o valor é o mesmo `Settings.PlayerName`).

**Roda do mouse em vez de setas ←/→.** O pedido permite cortar até "só a
roda" se um modelo de foco de teclado para a tela toda crescesse demais.
Cresceria: o resto da tela (menu, HUD) nunca teve conceito de foco de
teclado, e inventar um só para duas teclas em um controle seria mais
máquina nova do que o ganho justifica. `slider_input.go` implementa a roda
do mouse sobre a trilha; as setas ficam de fora.

## O que ficou de fora (e por quê)

- **Qualidade de efeitos das magias (alto/médio/baixo).** A desintegração
  dos monstros ganhou chave própria (28/09/2026); o resto continua aqui. Mexe no multiplicador de
  contagem de partícula de várias magias em `internal/skill`. O pedido exige
  ler `doc/skill_performance.md` antes — não foi lido nesta tarefa. Adiado
  para uma rodada que trate especificamente do orçamento de partículas.
- **Remapear teclas.** Exige uma tabela de binds inteira (tecla → ação,
  edição, conflito, persistência própria) — o pedido já sugere isso como
  fase 2 em vez de meio-feito.
- **Joystick fixo ou dinâmico.** "Nasce onde o dedo toca" muda a suposição
  de retângulo fixo que `TouchState.Update`/`input_handler.go` fazem hoje;
  cortado para não arriscar uma regressão de toque que nenhum teste aqui
  alcança (sem dispositivo Android neste ambiente).
- **Mostrar nomes dos jogadores.** Depende do nome do jogador estar na rede
  — ver "Decisões" acima.
- **Escala do HUD além da barra de vida.** `ui.SetHUDScale` existe e
  funciona, mas só `DrawHealthBar` o usa nesta rodada; estender a
  `ui/skill_hud.go`, `ui/wave_hud.go` etc. é mecânico e ficou para depois por
  tempo, não por dificuldade.

## Verificação manual (nada disto tem teste automatizado aqui)

Este ambiente não abre janela nem tem toque de verdade — só
`gofmt -l -e ./internal/` roda aqui. Precisa ser conferido no jogo, em
especial no celular:

- Arrastar cada slider com o DEDO (não só o mouse do desktop) — tocar na
  trilha salta pro valor, o degrau de 5% se sente, soltar toca a prévia.
- O botão de mudo por barramento preserva o valor ao desmutar.
- A engrenagem fica à esquerda de "Horda N/M" sem encostar no texto, e
  volta ao canto nos mapas sem horda (celular deitado e desktop).
- As cinco abas e o botão VOLTAR cabem sem se cortar numa tela de celular
  DEITADO bem baixa (a mais apertada testada só no papel foi ~360px de
  altura útil de painel — abaixo disso, conferir).
- O botão de mudo de cada barramento (Áudio) não estoura a borda direita do
  painel em telas estreitas — a linha ficou apertada de propósito para
  caber número + mudo na mesma largura de controle.
- Modo canhoto: joystick E cluster de ações trocam de lado, e as duas
  habilidades continuam orbitando para o lado de dentro da tela (não saem
  cortadas na borda esquerda).
- ESC dentro de uma partida abre Configurações; ESC de novo fecha; ESC
  enquanto espera um portal continua só cancelando o portal (não abre
  Configurações por cima).
- "Silenciar em segundo plano": minimizar o app no celular com música
  tocando e confirmar que ela para (e volta ao voltar pro app).
- VSync: mudar a opção, fechar e reabrir o jogo, confirmar que só reflete
  depois de reabrir (e que a tela avisa isso antes de fechar).
- Nome do jogador: digitar com teclado físico no Android (o virtual não
  está integrado nesta rodada — ver "Decisões").
