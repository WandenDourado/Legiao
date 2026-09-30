package entity

// De onde sai a pose de uma morte, e como ela vira uma explosao dentro do
// orcamento.

// deathPose do inimigo do HOST: o mesmo quadro, linha, espelho, angulo e
// ancora que Enemy.Draw acabou de usar.
func (e *Enemy) deathPose(def EnemyDef, ad EnemyAnimDef) DeathPose {
	p := DeathPose{Type: e.Type, Anim: e.Anim, Frame: ad.Column(e.AnimFrame), CX: e.Position.X, CY: e.Position.Y}
	switch def.Mode {
	case EnemyModeDirectional:
		p.Row = validEnemyRow(e.Row, ad.Rows)
		p.Mirror = e.Mirror
		p.CY -= enemyGroundOffset(ad, def.RenderScale)
	case EnemyModeFixed:
		p.CY += e.bossVisualAdvanceY(def) - enemyGroundOffset(ad, def.RenderScale)
	default:
		p.Angle = e.Angle
	}
	return p
}

// remoteDeathPose e a mesma coisa para o inimigo replicado, a partir do
// rastreador de animacao do cliente (DrawRemoteEnemy).
func remoteDeathPose(t EnemyType, def EnemyDef, ad EnemyAnimDef, st remoteEnemyAnim, x, y float32) DeathPose {
	p := DeathPose{Type: t, Anim: st.anim, Frame: ad.Column(st.frame), CX: x, CY: y}
	switch def.Mode {
	case EnemyModeDirectional:
		p.Row = validEnemyRow(st.row, ad.Rows)
		p.Mirror = st.mirror
		p.CY -= enemyGroundOffset(ad, def.RenderScale)
	case EnemyModeFixed:
		p.CY -= enemyGroundOffset(ad, def.RenderScale)
	default:
		p.Angle = st.angle
	}
	return p
}

// spawnDeathBurst transforma a pose numa explosao, respeitando o orcamento:
// grao normal se couber, grao dobrado se nao couber, e nada se nem assim
// couber — o monstro some como sempre sumiu, que e melhor que o jogo travar
// justo no pico da luta.
func spawnDeathBurst(pose DeathPose) {
	def := GetEnemyDef(pose.Type)
	tex, ad, ok := enemyTexture(def, pose.Anim)
	if !ok {
		return
	}
	cell := deathCellSrc(def, deathTuneFor(def))
	mask := deathMaskFor(ad.SpritePath, cell)
	if mask == nil {
		return // mascara ainda sendo varrida (death_fx_mask.go)
	}
	idx := pose.Row*mask.Cols + pose.Frame
	if idx < 0 || idx >= len(mask.Fine) {
		return
	}
	live := DeathFXFragments()
	cells, cellSrc := mask.Fine[idx], cell
	if live+len(cells) > deathFXBudget {
		cells, cellSrc = mask.Coarse[idx], cell*2
		if live+len(cells) > deathFXBudget {
			return
		}
	}
	if b := newAshBurst(pose, cells, cellSrc, tex, ad, def); b != nil {
		deathBursts = append(deathBursts, b)
	}
}
