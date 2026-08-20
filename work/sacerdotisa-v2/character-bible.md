# Sacerdotisa — v2 character bible

- **Character id:** `sacerdotisa`
- **Reference:** `C:/Users/guilh/Downloads/sacer.png`
- **Silhouette:** young human priestess with long wavy honey-blonde hair, a full ivory-white ceremonial robe, broad mantle, wide bell sleeves, central tabard, and ankle boots.
- **Palette:** ivory-white fabric; antique-gold filigree; muted blue-grey insets; blue chest jewel; warm light skin and honey-blonde hair.
- **Mirror-safe design:** centered chest jewel, cross tabard, belt ornament, and back panel; matched blue-grey robe panels and gold details on both sides. No one-sided sash, chain, pendant, hand prop, weapon, or hair accessory.
- **Scale and pivot:** visible silhouette must occupy 80–92% of a `128x192` frame (84% preferred for front/back); torso center at `x=64`; planted foot baseline at `y=186`; retain transparent margin for bounded normalization.
- **Walk contract:** frames 1–4 are the left-foot-forward half (contact, down, passing with left planted, up); frames 5–8 are the right-foot-forward half (contact, down, passing with right planted, up). Never repeat one leading foot across the full cycle.
- **Forbidden drift:** VFX, glow, aura, text, shadows, cropped parts, magenta on the character, unsymmetric details, duplicate limbs, baseline/torso movement, and scale changes.

## Direction attempts

| Direction | Decision | Reason | Report |
|---|---|---|---|
| S | accepted | Attempt 5; final rekey and normalization. Exact pivot/baseline, 85.9-87.0% height. | `review/S-final-pass2.json` |
| SW | accepted | Attempt 5; visual gait review confirmed opposite contacts. Exact pivot/baseline after final pass. **Correção 2026-08-03: frames `000`–`003` estavam espelhados (lote 1 invertido) e foram desespelhados; `004`–`007` são os corretos. Ver `attempt-manifest.md` — inclui a métrica calibrada que decide qual metade preservar.** | `review/SW-batch1-unflip-structural.json` |
| W | accepted | Attempt 5; second bounded normalization removed the final 1-2px anchor drift. | `review/W-final-pass2.json` |
| N | accepted | Attempt 2; attempt 1 exceeded the 24px safe-shift limit. Exact pivot/baseline. | `review/N-final-pass2.json` |
| NW | accepted | Attempt 1; rekeyed to remove a one-pixel matte residue, then normalized. | `review/NW-final-pass2.json` |
