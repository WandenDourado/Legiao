# UDP Host Discovery - Documentação Técnica

## Índice
1. [Visão Geral](#visão-geral)
2. [O Problema no Android](#o-problema-no-android)
3. [Solução: Query-Response UDP](#solução-query-response-udp)
4. [Multicast Lock no Android](#multicast-lock-no-android)
5. [Varredura TCP (Fallback)](#varredura-tcp-fallback)
6. [Protocolo de Descoberta](#protocolo-de-descoberta)
7. [Análise do Código](#análise-do-código)
8. [Fluxo de Conexão](#fluxo-de-conexão)
9. [Configuração do Android](#configuração-do-android)
10. [Considerações Técnicas](#considerações-técnicas)

---

## Visão Geral

O sistema de **UDP Host Discovery** permite que clientes encontrem automaticamente servidores (hosts) na rede local, eliminando a necessidade de o usuário digitar manualmente o endereço IP.

### Arquivos Envolvidos
| Arquivo | Responsabilidade |
|---------|-------------------|
| `internal/network/discovery.go` | Lógica de query-response UDP e TCP scan |
| `internal/ui/menu.go` | Interface de seleção de hosts descobertos |
| `cmd/android/build/android/AndroidManifest.xml` | Permissões Android |
| `internal/network/globals.go` | Armazena endereço do servidor para exibição |
| `cmd/android/build/android/build.gradle` | Configuração do Gradle |
| `cmd/android/build/gradle.properties` | Configuração AndroidX |

---

## O Problema no Android

### Comportamento Observado
- ✅ **Android como Host**: Consegue enviar broadcasts UDP, aparece no Desktop
- ❌ **Android como Cliente**: Não consegue listar hosts disponíveis

### Causa Raiz
O Android **consegue enviar** UDP broadcasts, mas **não consegue receber** broadcasts passivamente devido a restrições do sistema:
1. A partir do Android 10+, o sistema descarta pacotes UDP broadcast
2. O padrão de **consulta-resposta** contorna essa limitação

### Experiência Anterior (Ruim)
1. Usuário inicia o host no Desktop
2. Usuário precisa descobrir o IP (ex: `ipconfig`)
3. Usuário digita o IP manualmente no cliente Android
4. Se errar o IP, precisa tentar novamente

### Solução Atual (04/05/2026)
- ❌ Código Java/MulticastLock **removido** (causava crash `ClassNotFoundException`)
- ✅ Padrão **Query-Response UDP** funciona sem MulticastLock
- ✅ **Sem código Java**: `hasCode="false"` no AndroidManifest.xml
- ✅ Corrigido bug do `ticker` em `StartQuerySender()` (ticker movido para dentro da goroutine)

---

## Solução: Query-Response UDP

Implementamos um padrão de **consulta-resposta** que contorna a restrição de recebimento de broadcasts no Android.

### Por que funciona?
- O Android **consegue enviar** pacotes UDP (mesmo broadcast)
- O Android **consegue receber** respostas diretas (unicast)
- O host responde **diretamente** para o cliente (não é broadcast)

### Como Funciona

1. **Cliente envia consulta**: `LEGION_QUERY:porta_local` via broadcast UDP
2. **Host recebe e responde**: `LEGION_RESPONSE:0.0.0.0:9000` diretamente para o cliente
3. **Cliente recebe resposta** e extrai o IP do host

### Vantagens
- ✅ Funciona no Android (não precisa "escutar" broadcast passivamente)
- ✅ Resposta direta (mais confiável)
- ✅ Não precisa de entrada manual de IP
- ✅ Funciona também no Desktop

### Diagrama
```
┌─────────────┐                    ┌─────────────┐
│    HOST     │                    │   CLIENT    │
│  (Desktop  │                    │  (Android)  │
│   ou       │                    │             │
│   Android) │                    │             │
└──────┬──────┘                    └──────┬──────┘
       │                                  │
       │  ←── QUERY (broadcast) ───────── │
       │     "LEGION_QUERY:12345"         │
       │                                  │
       │  ─── RESPONSE (direta) ──────> │
       │     "LEGION_RESPONSE:192.168.1.42:9000" │
       │                                  │
       │                                  │ 3. Extrai IP: 192.168.1.42
       │                                  │ 4. Conecta via TCP
```

### Código: `StartQuerySender` (Cliente)
```go
func StartQuerySender(gamePort int)
```
- Cria socket UDP temporário na porta 0 (sistema escolhe)
- Envia `LEGION_QUERY:porta_local` via broadcast a cada 3 segundos
- Escuta respostas em `receiveQueryResponses()`

### Código: `StartQueryResponder` (Host)
```go
func StartQueryResponder(gamePort int, stopChan chan struct{})
```
- Escuta na porta 9001
- Ao receber `LEGION_QUERY`, responde `LEGION_RESPONSE:0.0.0.0:9000`
- IP real é substituído pelo de origem no cliente

---

## Configuração do Android

### Estrutura de Arquivos
```
cmd/android/build/
├── android/
│   ├── AndroidManifest.xml          # Permissões e configuração
│   ├── build.gradle                # Configuração do Gradle
│   └── res/                        # Resources (icons, etc.)
├── build.gradle                   # Gradle root
├── gradle.properties              # AndroidX config
└── main.go                       # Ponto de entrada Go
```

### Permissões no `AndroidManifest.xml`
```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.ACCESS_WIFI_STATE" />
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
<uses-permission android:name="android.permission.CHANGE_WIFI_STATE" />
<uses-permission android:name="android.permission.CHANGE_WIFI_MULTICAST_STATE" />
```

### Correção de Crash (04/05/2026)
- **Problema**: Código Java (`LegiaoApp`, `MulticastLockHelper`) causava `ClassNotFoundException`
- **Solução**: Removido código Java, voltamos para `hasCode="false"`
- **Motivo**: O padrão query-response funciona sem MulticastLock (respostas são unicast, não broadcast)
- **Status**: ✅ Android funciona sem código Java

### `gradle.properties`
```
org.gradle.jvmargs=-Xmx1536m
android.useAndroidX=true
android.enableJetifier=true
android.suppressUnsupportedCompileSdk=35
```

---

## Varredura TCP (Fallback)

Se o query-response falhar, implementamos varredura TCP na sub-rede como fallback.

### Como Funciona
```go
func StartTCPScan(gamePort int)
```

1. **Detecta a sub-rede local** (ex: `192.168.1.*`)
2. **Testa todas as IPs** de 1 a 254 na porta 9000
3. **Paralelismo controlado**: Usa semáforo com 20 conexões simultâneas
4. **Timeout curto**: 300ms por tentativa

### Vantagens
- ✅ Funciona no Android (não usa UDP)
- ✅ Não precisa de permissões especiais
- ✅ Encontra hosts que estejam ativos

### Desvantagens
- ⚠️ Mais lento que UDP (2-3 segundos)
- ⚠️ Gera tráfego na rede (254 tentativas)
- ⚠️ Firewall pode bloquear conexões TCP externas

---

## Protocolo de Descoberta

### Formato das Mensagens

#### Query (Cliente → Broadcast)
```
LEGION_QUERY:<PORTA_LOCAL>
```

Exemplo:
```
LEGION_QUERY:12345
```

#### Response (Host → Cliente, direto)
```
LEGION_RESPONSE:<IP>:<PORTA>
```

Exemplo:
```
LEGION_RESPONSE:192.168.1.42:9000
```

### Campos da Response
| Campo | Descrição | Exemplo |
|-------|-----------|--------|
| Prefixo | Identifica resposta do jogo | `LEGION_RESPONSE` |
| IP | Endereço IP do host (0.0.0.0 é substituído) | `192.168.1.42` |
| Porta | Porta TCP onde o jogo está escutando | `9000` |

### Portas Utilizadas
| Porta | Protocolo | Uso |
|-------|-----------|-----|
| 9000 | TCP | Comunicação do jogo (estado, inputs) |
| 9001 | UDP | Descoberta de hosts (query-response) |

---

## Análise do Código

### 1. `internal/network/discovery.go`

#### Constantes
```go
const (
    DiscoveryPort = 9001                // Porta UDP para query-response
    DiscoveryInterval = 3 * time.Second  // Intervalo entre queries
    TCPScanTimeout = 300 * time.Millisecond
)
```

#### Função: `StartQuerySender` (Cliente)
```go
func StartQuerySender(gamePort int)
```
**O que faz:** Envia consultas UDP e escuta respostas

**Passo a passo:**
1. Cria socket UDP na porta 0 (sistema escolhe uma livre)
2. Envia `LEGION_QUERY:porta_local` via broadcast a cada 3s
3. Escuta respostas em `receiveQueryResponses()`
4. Para quando `stopChan` é fechado

#### Função: `StartQueryResponder` (Host)
```go
func StartQueryResponder(gamePort int, stopChan chan struct{})
```
**O que faz:** Responde a consultas dos clientes

**Passo a passo:**
1. Escuta na porta 9001
2. Ao receber `LEGION_QUERY`, extrai a porta de resposta do cliente
3. Responde `LEGION_RESPONSE:0.0.0.0:9000` diretamente para o cliente
4. IP real é substituído pelo de origem no cliente

#### Função: `StartTCPScan` (Fallback)
```go
func StartTCPScan(gamePort int)
```
Varredura TCP na sub-rede para hosts que estejam escutando na porta 9000.

---

### 2. `internal/ui/menu.go`

#### Iniciar Discovery (no Join)
```go
// Start query sender for clients (UDP query-response works on Android)
network.StartQuerySender(9000)
```

#### Exibir Hosts Descobertos
```go
hosts := network.GetDiscoveredHosts()

if len(hosts) == 0 {
    rl.DrawText("Searching for hosts...", ...)
} else {
    for i, host := range hosts {
        // Desenha botão para cada host encontrado
    }
}
```

#### Três Modos de Conexão
1. **Automático (Query-Response)**: Funciona sozinho após clicar "Join Game"
2. **Scan TCP**: Botão para varredura TCP (fallback manual)
3. **Manual IP**: Entrada manual de IP (último recurso)

---

## Fluxo de Conexão

### Diagrama Passo a Passo

```
┌─────────────┐                    ┌─────────────┐
│    HOST     │                    │   CLIENT    │
└──────┬──────┘                    └──────┬──────┘
       │                                  │
       │ 1. Inicia TCP server (:9000)     │
       │ 2. Inicia Query Responder (:9001)│
       │                                  │
       │                                  │ 3. Clica "Join Game"
       │                                  │ 4. StartQuerySender()
       │                                  │
       │  ←── QUERY (broadcast) ───────── │ 5. A cada 3s
       │     "LEGION_QUERY:12345"         │
       │                                  │
       │  ─── RESPONSE (direta) ──────> │ 6. Host responde
       │     "LEGION_RESPONSE:IP:9000"    │ 7. Cliente extrai IP
       │                                  │ 8. Host aparece na lista
       │                                  │
       │                                  │ 9. Usuário clica no host
       │  <──── CONEXÃO TCP ──────────── │
       │         (IP:9000)               │ 10. Envia MsgJoin
       │                                  │
       │ 11. Aceita conexão               │
       │ 12. Registra player             │
       │ 13. BroadcastStateUpdate()      │
       │  ─── MsgStateUpdate ─────────> │
       │                                  │ 14. Atualiza RemotePlayers
       │                                  │ 15. Começa o jogo!
```

### Passos Detalhados

#### Host:
1. Clica em "Host Game"
2. TCP server sobe na porta 9000
3. Query Responder inicia na porta 9001
4. Broadcast UDP inicia (para compatibilidade com Desktop)
5. `ServerAddress` é definido (ex: `192.168.1.42:9000`)

#### Cliente (Android):
1. Clica em "Join Game"
2. `StartQuerySender(9000)` é chamado
3. Envia queries via broadcast, recebe respostas diretas
4. Hosts aparecem na lista automaticamente
5. Se nada aparecer após alguns segundos:
   - Clica "Scan TCP" (fallback)
   - Ou "Manual IP" (último recurso)
6. Clica no host desejado e conecta via TCP

---

## Configuração do Android

### Estrutura de Arquivos
```
cmd/android/build/
├── android/
│   ├── AndroidManifest.xml          # Permissões e configuração
│   ├── build.gradle                # Configuração do Gradle
│   └── src/main/java/com/legiao/app/
│       ├── MulticastLockHelper.java # Adquire MulticastLock
│       └── LegiaoApp.java          # Application class
├── build.gradle                   # Gradle root
└── main.go                       # Ponto de entrada Go
```

### Passos para Compilar
1. **Compile o código Go**:
   ```bash
   cd cmd/android/build
   ./androidcompile.bat
   ```

2. **Gere o APK**:
   ```bash
   gradlew.bat assembleDebug
   ```

3. **Instale no dispositivo**:
   ```bash
   adb install android/build/outputs/apk/debug/app-debug.apk
   ```

---

## Considerações Técnicas

### Por que UDP Query-Response e não apenas TCP?

| UDP Query-Response | TCP Scan |
|-----|-----|
| Muito rápido (resposta em ms) | Lento (2-3 segundos) |
| Baixo tráfego (1 pacote a cada 3s) | Alto tráfego (254 tentativas) |
| Automático | Manual (botão "Scan TCP") |
| Funciona em tempo real | Demora para encontrar |

### Segurança
- Apenas hosts na **mesma rede local** são descobertos
- O prefixo `LEGION_` evita processar pacotes de outros programas
- Porta 9001 pode ser bloqueada no firewall se desejado

### Limitações
- **Não funciona na internet:** Apenas redes locais (LAN/Wi-Fi)
- **Redes complexas:** Se houver múltiplos roteadores/sub-redes, pode não funcionar
- **Firewall:** O host deve permitir entrada TCP na porta 9000

### Troubleshooting

#### 1. Host não aparece no Android:
   - ✅ Verifique se host e cliente estão na mesma rede Wi-Fi
   - ✅ Desative VPNs (podem bloquear broadcasts)
   - ✅ Verifique se o MulticastLock foi adquirido (veja o log `LegiaoMulticast`)
   - ✅ Tente o botão "Scan TCP" como fallback

#### 2. Múltiplos hosts:
   - Todos aparecem na lista
   - Usuário escolhe qual entrar

#### 3. Verificando logs no Android:
   ```bash
   adb logcat | findstr Legiao
   adb logcat | findstr LegiaoMulticast
   ```

---

## Resumo

O sistema de descoberta de hosts foi completamente reformulado para funcionar no Android:

### Implementado
1. ✅ **Query-Response UDP**: Cliente envia consulta, host responde diretamente
2. ✅ **Multicast Lock**: Código Java para adquirir lock no Android
3. ✅ **Permissões completas**: Todas as 5 permissões de rede no Manifest
4. ✅ **TCP Scan Fallback**: Varredura TCP caso query-response falhe
5. ✅ **Zero configuração manual**: Usuário apenas clica no host desejado

### Arquivos Modificados/Criados
| Arquivo | Alteração |
|--------|-----------|
| `internal/network/discovery.go` | Adicionado `StartQuerySender`, `StartQueryResponder`, `StartTCPScan` |
| `internal/ui/menu.go` | Usa `StartQuerySender()` em vez de listener passivo |
| `cmd/android/build/android/AndroidManifest.xml` | Permissões + `hasCode="true"` + `android:name=".LegiaoApp"` |
| `cmd/android/build/android/src/main/java/.../MulticastLockHelper.java` | (Novo) Classe para MulticastLock |
| `cmd/android/build/android/src/main/java/.../LegiaoApp.java` | (Novo) Application class |
| `cmd/android/build/android/build.gradle` | Adicionado `java.srcDirs` e dependência |
| `doc/udp_discovery.md` | Esta documentação |

### Como Testar
1. **Desktop como Host**: Execute o jogo, clique "Host Game"
2. **Android como Cliente**: Abra o app, clique "Join Game"
3. **Resultado**: Host deve aparecer na lista automaticamente após ~3 segundos
4. **Fallback**: Se não aparecer, clique "Scan TCP"
