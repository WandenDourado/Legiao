package entity

// Contadores de recurso do pacote, para o medidor do F3.
//
// Eles existem por uma razao so: um numero que SO CRESCE entre fases e a
// assinatura de um vazamento, e sem alguem publicando o numero a unica forma
// de saber era ler o codigo e supor. As folhas de inimigo sao um cache de
// sessao de proposito (o registro e estatico e `PreloadEnemyTextures` sobe
// todas de uma vez), entao aqui o esperado e um numero PARADO; os trackers de
// animacao remota sao por inimigo vivo e podados por TTL, entao o esperado e
// um numero que sobe e desce com a horda.
//
// Qualquer um dos dois subindo fase apos fase e defeito, nao uso.

// EnemyTextureCacheSize e quantas folhas de inimigo estao na VRAM.
func EnemyTextureCacheSize() int {
	enemyTexMu.Lock()
	defer enemyTexMu.Unlock()
	return len(enemyTextures)
}

// RemoteAnimCount e quantos inimigos remotos tem tracker de animacao vivo no
// cliente. Ver pruneRemoteAnimsLocked em enemy_sprite.go.
func RemoteAnimCount() int {
	remoteAnimMu.Lock()
	defer remoteAnimMu.Unlock()
	return len(remoteAnims)
}
