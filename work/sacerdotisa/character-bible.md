# Sacerdotisa — character bible

- **Character id:** `sacerdotisa`
- **Source reference:** `C:/Users/guilh/Downloads/sacer.png`
- **Role:** young human priestess; no weapon or handheld prop.
- **Silhouette:** long, ankle-length ceremonial robe with a broad shoulder cape, very wide bell sleeves, layered front tabard, narrow waist belt, and low heeled ankle boots. Long wavy hair falls over both shoulders and behind the cape.
- **Palette:** ivory-white fabric; warm antique-gold trim, filigree, crosses, and jewelry; muted blue-grey lining, belt panels, and gemstone settings; light honey-blonde hair; warm light skin.
- **Identity details:** gold-trimmed ivory mantle closes at the chest with a blue oval jewel; ornate gold-and-blue belt with an asymmetrical blue-grey sash and dangling gold medallion on **body-right**; central long tabard carries a gold cross; blue-grey robe insets remain beneath the white overskirt.
- **Body-side asymmetry:** the gemmed sash, chains, and hanging ornament always remain on body-right. Do not mirror source art for a direction if it moves that detail to body-left.
- **Animation treatment:** poised, modest walk. Robe hem, wide cuffs, hair ends, belt sash, and pendant lag one pose behind the hips. Hands empty and close to a calm walking swing.
- **Scale / anchor:** centered full body with generous side padding; boots contact a single shared baseline.
- **Forbidden drift:** no magic cross, glow, particles, aura, staff, book, wand, halo, weapon, text, costume color changes, bare arms, shortened dress, moved body-right ornament, duplicated limbs, shadows, borders, or magenta on the character.

## Direction audit

| Direction | Decision | Reason | Review report |
|---|---|---|---|
| S | accepted | Eight-pose front walk; alpha, matte, and baseline checks passed. | `review/S.json` |
| SW | accepted | Three-quarter front walk; alpha, matte, and baseline checks passed. | `review/SW.json` |
| W | accepted | Third source grid passed after two rejected attempts with anchor drift above 4 px. | `review/W.json` |
| N | accepted | Back walk; alpha, matte, and baseline checks passed. | `review/N.json` |
| NW | accepted | Three-quarter back walk; narrow torso anchor range (`30%–46%`) produced stable anchors without clipping. Three earlier source grids were rejected. | `review/NW.json` |

## Model-sheet decision

The five-view model sheet (`model/five-view-model-sheet.png`) was visually approved as the identity reference before animation: front, left three-quarter/back, left profile, back, and right three-quarter/back all retain the robe, palette, hair, and body-right ornament.
