package network

// O que NAO precisa ir no fio a cada tique.
//
// Um snapshot carregava, sessenta vezes por segundo, campos que nao mudam
// desde o join: a cor e o personagem de cada jogador, o tipo e a cor de cada
// inimigo, a vida maxima dos dois. Num PlayerState de 231 bytes isso e ~70;
// num EnemyState de 125 sao ~45, e o world_03 manda 83 inimigos por tique.
//
// A regra que sai daqui: **identidade e anunciada uma vez, estado vai a cada
// tique**. Quem recebe guarda a identidade num cache por ID e recompoe o
// snapshot completo, entao nada acima da camada de rede percebe a diferenca —
// RemotePlayers e RemoteEnemies continuam tendo todos os campos preenchidos.
//
// Isso muda uma regra escrita em doc/network.md: o cliente deixou de tratar o
// snapshot como substituicao pura. Ele ainda SUBSTITUI a lista (quem sumiu
// sumiu), mas COMPLETA os campos de identidade que vieram vazios. A diferenca
// importa porque lista vazia continua significando "limpe tudo".

// slimPlayer remove do PlayerState o que o receptor ja sabe.
//
// MaxHealth entra aqui e Health nao: a vida maxima e do personagem e nao muda,
// a atual muda toda hora. IsDead e RespawnIn tambem ficam, por serem estado.
func slimPlayer(p PlayerState) PlayerState {
	p.Color = ""
	p.Character = ""
	p.MaxHealth = 0
	return p
}

// slimEnemy remove do EnemyState o que o receptor ja sabe.
func slimEnemy(e EnemyState) EnemyState {
	e.Type = ""
	e.Color = ""
	e.MaxHealth = 0
	return e
}

// identityCache reconstitui o que o emissor omitiu.
//
// Mora no receptor (cliente) e e alimentado por todo snapshot que TRAZ
// identidade — o primeiro de cada entidade, e o snapshot completo que um
// cliente recebe ao entrar. Um campo que chega vazio e "igual ao ultimo que
// voce viu", nunca "apagou".
type identityCache struct {
	players map[string]PlayerState
	enemies map[string]EnemyState
}

func newIdentityCache() *identityCache {
	return &identityCache{
		players: map[string]PlayerState{},
		enemies: map[string]EnemyState{},
	}
}

// restorePlayers completa a lista recebida e atualiza o cache.
func (c *identityCache) restorePlayers(in []PlayerState) []PlayerState {
	out := make([]PlayerState, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, p := range in {
		known := c.players[p.PlayerID]
		if p.Color == "" {
			p.Color = known.Color
		}
		if p.Character == "" {
			p.Character = known.Character
		}
		if p.MaxHealth == 0 {
			p.MaxHealth = known.MaxHealth
		}
		c.players[p.PlayerID] = PlayerState{
			PlayerID:  p.PlayerID,
			Color:     p.Color,
			Character: p.Character,
			MaxHealth: p.MaxHealth,
		}
		seen[p.PlayerID] = true
		out = append(out, p)
	}
	// Quem saiu do snapshot saiu do jogo; guardar a identidade dele para
	// sempre seria vazamento lento numa sessao longa.
	for id := range c.players {
		if !seen[id] {
			delete(c.players, id)
		}
	}
	return out
}

// restoreEnemies faz o mesmo pelos inimigos. A poda importa mais aqui: uma
// corrida de hordas cria e destroi centenas de IDs.
func (c *identityCache) restoreEnemies(in []EnemyState) []EnemyState {
	out := make([]EnemyState, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, e := range in {
		known := c.enemies[e.EnemyID]
		if e.Type == "" {
			e.Type = known.Type
		}
		if e.Color == "" {
			e.Color = known.Color
		}
		if e.MaxHealth == 0 {
			e.MaxHealth = known.MaxHealth
		}
		c.enemies[e.EnemyID] = EnemyState{
			EnemyID:   e.EnemyID,
			Type:      e.Type,
			Color:     e.Color,
			MaxHealth: e.MaxHealth,
		}
		seen[e.EnemyID] = true
		out = append(out, e)
	}
	for id := range c.enemies {
		if !seen[id] {
			delete(c.enemies, id)
		}
	}
	return out
}

// announcer lembra a quem o host ja contou a identidade de cada inimigo.
//
// Vive no host e e um conjunto so, e nao um por peer, porque o broadcast e
// para todos ao mesmo tempo. O cliente que entra no meio da partida e o caso
// que isso NAO cobre, e ele e resolvido a parte: sendStateToClient manda um
// snapshot completo so para ele.
type announcer struct {
	enemies map[string]bool
}

func newAnnouncer() *announcer { return &announcer{enemies: map[string]bool{}} }

// wireEnemies devolve a lista para transmitir: inteira para quem e novo,
// magra para quem ja foi anunciado. Poda quem sumiu no mesmo passe.
func (a *announcer) wireEnemies(in []EnemyState) []EnemyState {
	out := make([]EnemyState, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, e := range in {
		seen[e.EnemyID] = true
		if a.enemies[e.EnemyID] {
			out = append(out, slimEnemy(e))
			continue
		}
		a.enemies[e.EnemyID] = true
		out = append(out, e)
	}
	for id := range a.enemies {
		if !seen[id] {
			delete(a.enemies, id)
		}
	}
	return out
}

// forget zera o que ja foi anunciado. Chamado quando a fase reinicia ou o
// mapa troca: os IDs antigos morreram, e um ID reaproveitado precisa ser
// anunciado de novo.
func (a *announcer) forget() { a.enemies = map[string]bool{} }
