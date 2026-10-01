package skill

// O ritmo do jogo, espelhado para este pacote.
//
// entity.GameSpeed (internal/entity/gamespeed.go) e a fonte; aqui ele ganha um
// nome local para que as constantes de magia o usem sem que cada um dos seis
// arquivos de projetil precise importar `entity` so por causa de um numero.
//
// O QUE LEVA O FATOR NESTE PACOTE: velocidade de projetil e de unidade
// invocada, e os TTL que existem como rede de seguranca proporcional ao voo.
//
// O QUE NAO LEVA: os ALCANCES (ArrowRange, FireballRange, CelestialRange...).
// Aqueles sao distancias, nao velocidades — o projetil chega mais rapido ao
// mesmo lugar, e e exatamente isso que se quis. Tambem nao levam recarga,
// dano, raio de explosao nem duracao de area: ver a lista do "deliberadamente
// nao toca" em entity/gamespeed.go.

import "github.com/WandenDourado/Legiao/internal/entity"

const gameSpeed = entity.GameSpeed
