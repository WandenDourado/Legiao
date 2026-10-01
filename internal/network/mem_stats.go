package network

// O tamanho do mundo espelhado, para o medidor do F3.
//
// Os tres mapas globais sao substituidos inteiros a cada snapshot no cliente e
// podados no host (pruneRemotePlayersLocked), entao o esperado e um numero que
// acompanha o que esta em campo. Um deles crescendo fase apos fase seria uma
// poda que parou de acontecer — e a unica forma de ver isso de fora e
// publicando a contagem.

// MirrorCounts e quantas entidades o espelho de rede guarda agora.
type MirrorCounts struct {
	Players     int
	Enemies     int
	Projectiles int
}

// SpecterCount e quantos espectros da Legiao estao em campo nesta maquina,
// seja ela host ou cliente. Zero quando ninguem conjurou a suprema do
// Necromante. Ver skill.Manager.SpecterCount para por que este numero merece
// espaco no painel.
func SpecterCount() int {
	if CurrentHost != nil && CurrentHost.Skills != nil {
		return CurrentHost.Skills.SpecterCount()
	}
	if ClientSkills != nil {
		return ClientSkills.SpecterCount()
	}
	return 0
}

// Mirror devolve as contagens do mundo espelhado.
func Mirror() MirrorCounts {
	var c MirrorCounts

	RemotePlayersMutex.Lock()
	c.Players = len(RemotePlayers)
	RemotePlayersMutex.Unlock()

	RemoteEnemiesMutex.Lock()
	c.Enemies = len(RemoteEnemies)
	RemoteEnemiesMutex.Unlock()

	RemoteProjectilesMutex.Lock()
	c.Projectiles = len(RemoteProjectiles)
	RemoteProjectilesMutex.Unlock()

	return c
}
