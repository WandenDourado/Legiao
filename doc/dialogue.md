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
| `trigger` | `on_map_start` ou `on_waves_cleared`. Um roteiro por gatilho, por mapa. |
| `speaker` | Nome exibido. Vazio = narração sem nome. |
| `portrait` | Chave de personagem (`mago`, `paladina`, `sacerdotisa`, `arqueiro`, `necromante`). Vazio = sem imagem. |
| `text` | A fala. Quebra de linha é automática, escreva o parágrafo inteiro. |

Editar texto não exige recompilar: o arquivo é lido em runtime, como os mapas.

## Gatilhos

| Gatilho | Quando dispara |
|---|---|
| `on_map_start` | Primeiro quadro em que o mapa está no ar (inclusive depois de um portal). |
| `on_waves_cleared` | Quando `WaveState.Phase` vira `cleared`, ou seja, a última horda do mapa morreu. |

Cada roteiro toca uma vez por sessão, guardado pelo `id`. Voltar para um mapa não repete a cena.

## Retratos

O retrato vem do `reference.png` do personagem (`CharacterDef.ReferenceImagePath`), não de uma pasta de retratos: quando a arte do personagem muda, a cena acompanha sem reescrever roteiro.

Três casos desenham **sem** imagem:

1. `portrait` vazio — narração, voz de fora, explicação.
2. Chave que não existe no registro de personagens (erro de digitação). Sem essa checagem, `entity.GetCharacter` devolveria o Mago como fallback e a fala sairia com o rosto errado.
3. Personagem listado em `placeholderPortraits` (`internal/ui/dialogue_box.go`) — arte ainda emprestada de outro personagem. Hoje: **Necromante**, que reusa a folha do Mago. Apague a linha do mapa quando a arte própria existir; o roteiro já pede `"portrait": "necromante"`.

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

O envio é por mudança, não por quadro, porque o broadcast periódico do host mora dentro de `UpdateSimulation` — exatamente o que a cena congela. Quem entra no meio de uma cena recebe o estado na hora do join (`sendDialogueTo`).

## Arquivos

| Arquivo | Responsabilidade |
|---|---|
| `internal/dialogue/script.go` | Tipos do roteiro (dado puro, sem rede e sem desenho). |
| `internal/dialogue/load.go` | Leitura e validação do JSON do mapa. |
| `internal/dialogue/runner.go` | Cursor sobre uma cena em execução. |
| `internal/game/dialogue.go` | Quem dispara, quem avança, quem encerra (host/solo). |
| `internal/network/dialogue.go` | Estado compartilhado e publicação para os peers. |
| `internal/ui/dialogue_box.go` | Caixa, retrato, quebra de texto, dica de comando. |

## Limitações conhecidas

- Um roteiro por gatilho por mapa. Cena maior = mais linhas, não mais roteiros.
- Não há escolha de resposta nem ramificação: a cena é linear.
- Não há efeito de máquina de escrever; a linha aparece inteira.
- O texto usa a fonte padrão do raylib. Se algum acento aparecer como bloco no jogo, a saída é embutir uma fonte com `LoadFontEx` — não tirar os acentos do JSON.


## Gatilhos

| Gatilho | Quando | Da corrida? |
|---|---|---|
| `on_map_start` | Primeiro quadro do mapa no ar | não (é do mapa) |
| `on_last_stand` | **Durante** a luta, quando o grupo está caindo | sim |
| `on_waves_cleared` | Última horda morta | sim |

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
