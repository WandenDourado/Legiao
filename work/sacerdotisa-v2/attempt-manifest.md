# Sacerdotisa v2 attempt manifest

Each direction is an independent worker output. Only the final paths below were copied to `work/sacerdotisa-v2/frames/` and assembled.

| Direction | Attempts / corrections | Final decision | Accepted source | Final checks |
|---|---:|---|---|---|
| S | 5 attempts; final matte rekey | accepted | `attempts/S/attempt-5/frames-final-pass2/S` | x=64; y=186; 85.9-87.0% height; clear border; no magenta |
| SW | 5 attempts; prior gait rejected; **correção 2026-08-03: lote 2 desespelhado** | accepted | `attempts/SW/attempt-5/frames-final-pass2/SW` + flip horizontal de `004`–`007` | x=64; y=186; 86.5-87.5% height; contacts 000/004 alternate; `orientation pass` |
| W | 4 recorded attempts; final second normalization | accepted | `attempts/W/attempt-5/frames-final-pass2/W` | x=64; y=186; clear border; no magenta |
| N | 2 attempts | accepted | `attempts/N/attempt-2/frames-final-pass2/N` | x=64; y=186; 81.8-84.9% height; attempt 1 rejected for unsafe shift |
| NW | 1 attempt; one rekey correction | accepted | `attempts/NW/attempt-1/frames-final-pass2/NW` | x=64; y=186; clear border; no magenta |

The five canonical rows are `S, SW, W, N, NW`. The game mirrors `W` for `E`, `SW` for `SE`, and `NW` for `NE`; no redundant east-facing source images were generated.

## Correção 2026-08-03 — SW, lote 2 invertido

**Defeito.** Os frames `004`–`007` da linha SW estavam espelhados horizontalmente em relação a `000`–`003` (o defeito "batch 2 invertido" descrito no SKILL). Em jogo, a Sacerdotisa começava a andar na direção certa e invertia na segunda metade do ciclo.

**Evidência.**

- `structural_check` da linha SW: `orientation: fail`, `flipped_frames: [5, 6, 7]`, `hard_fail: true`. Foi a única das 5 linhas da Sacerdotisa e das 15 linhas de mago/paladina/arqueiro a reprovar.
- Marcador fixo no corpo (centroide do ouro − centroide do teal, na faixa da saia, y 118–186): frames 0–3 = `+2.8 +5.4 +7.2 +5.6`; frames 4–7 = `-4.8 -4.2 -5.9 -0.6`. Troca de sinal exatamente na fronteira do lote. Linha W (controle) fica em `+10` nos oito frames.
- Correlação por faixa entre o frame `i` e o `i+4`: na barra da saia (região fixa no corpo, imune ao balanço dos braços) `same` 0.61–0.64 contra `mirror` 0.84–0.92.

**Qual metade estava certa: a segunda (4–7).**

> ⚠️ Na primeira passada desta correção foi concluído o oposto — que a metade certa era 0–3 — e os frames `004`–`007` foram espelhados. Isso deixou a linha SW internamente consistente porém **inteira virada para o lado errado**, e o defeito foi reportado em jogo: andando na diagonal baixo-esquerda a personagem aparecia virada para baixo-direita. O erro foi de leitura visual do model sheet, sem métrica calibrada. Registrado aqui de propósito para que a métrica abaixo seja usada em vez do olho.

**Métrica calibrada (a que decide).** Posição do ouro da saia (faixa y 55–78% da figura, que contém a cruz do tabardo e exclui o galão da barra) em relação ao centro da silhueta, em % da largura da figura. Negativo = tabardo à esquerda = virada para a esquerda do observador.

| | model sheet | linha do jogo (média) |
|---|---:|---:|
| S | −0.3 | −1.2 |
| SW | **−7.4** | **−8.1** (após a correção) |
| W | −31.6 | −10.0 |
| N | −0.5 | +0.1 |
| NW | +6.9 | +1.0 |

Todas as cinco linhas têm o mesmo sinal do model sheet, e SW cai praticamente em cima do valor de referência. No estado original a linha SW estava partida: frames 0–3 = `+6.7 +11.1 +10.7 +7.3` (sinal errado) e frames 4–7 = `−4.1 −8.7 −7.4 −8.9` (sinal certo, média −7.3 contra −7.4 do model). Ou seja, **o lote invertido era o 1, não o 2**.

**Origem.** O defeito já aparece em `attempts/SW/attempt-5/sliced/`, ou seja, veio da geração e não do pós-processamento. As tentativas 1, 2 e 4 passam no `orientation`, mas foram rejeitadas por gait, então não serviam como substitutas.

**Correção final.** Flip horizontal dos frames `000`–`003`; `004`–`007` mantidos como gerados. Sheet remontada com `build_sheet.py` e implantada em `assets/sprites/sacerdotisa/`.

**Resultado.** `orientation` passou de `fail [5,6,7]` para `pass []`; continuidade na emenda 3→4 subiu de 0.744 para 0.920; `duplicates`, `height` e `preflight_renderer` ok; baseline 186 e torso_x 61.8–63.0 nos oito frames (mais uniforme que o estado original, que misturava ~65 no lote 1 com ~63 no lote 2; a linha W já aceita ships em 61.2–61.8 pela mesma medida). Backup do estado original em `work/sacerdotisa-v2/backup-sw-batch2/`.

**Nota sobre o `structural_check`.** O gate detecta que as duas metades discordam, mas **não diz qual delas está certa** — a correlação dele é tolerante a espelho. Ele apontou `flipped_frames: [5,6,7]` simplesmente porque compara a segunda metade contra a primeira, tomando a primeira como referência. Para decidir qual metade preservar é obrigatório usar uma métrica calibrada contra o model sheet, como a tabela acima.

**Pendência conhecida, não relacionada.** A linha `N` reprova em `height_consistency`: alturas `163 162 158 161 / 157 157 158 157`, spread 6px contra o limite de 5, com as duas metades em patamares diferentes. O gate sugere `scale_fit --harmonize-halves`. Não foi tocada nesta correção.
