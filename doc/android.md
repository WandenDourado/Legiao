# Configuração do Ambiente e Execução no Android

Este documento descreve como configurar o ambiente e compilar o projeto **Legião** para dispositivos Android.

## Pré-requisitos

1.  **Go (Golang)**: Instalado e configurado no PATH.
2.  **Android SDK**: Necessário para ferramentas de build.
3.  **Android NDK**: Versão recomendada: `25.2.9519653`.
    *   *Nota*: Versões muito recentes (como NDK 27) podem não ser compatíveis com o `raylib-go` no momento.
4.  **C-Compiler para Android**: O script utiliza o `clang` contido no NDK.

## Estrutura de Arquivos para Android

*   `cmd/android/main.go`: Ponto de entrada otimizado para Android (usa `rl.SetMain`).
*   `cmd/android/build/`: Pasta contendo o ambiente de compilação.
    *   `android/AndroidManifest.xml`: Configurações do aplicativo (nome, permissões, orientação).
    *   `androidcompile.bat`: Script de automação para compilação CGO cruzada.

## Como Compilar

1.  Abra o arquivo `cmd/android/build/androidcompile.bat`.
2.  Ajuste as variáveis de ambiente no topo do arquivo para apontar para as suas pastas locais (especialmente `ANDROID_HOME` e `ANDROID_NDK_HOME`).
3.  Abra um terminal na pasta `cmd/android/build/`.
4.  Execute o script:
    ```bash
    ./androidcompile.bat
    ```
5.  O script gerará arquivos `.so` na pasta `android/libs/` para as arquiteturas selecionadas.

## Como Gerar o APK

Após compilar as bibliotecas nativas, você pode usar o Gradle para gerar o APK:
1.  Navegue até `cmd/android/build/`.
2.  Execute:
    ```bash
    ./gradlew assembleDebug
    ```
3.  O APK será gerado em `cmd/android/build/android/build/outputs/apk/debug/`.

## Observações Técnicas

*   **Resolução**: No Android, o jogo inicia automaticamente em tela cheia usando `rl.InitWindow(0, 0, "Legião")`.
*   **Controles**: O jogo utiliza um `VirtualJoystick` na tela para processar entradas de toque.
*   **Compatibilidade**: O código principal reside em `internal/`, garantindo que a lógica seja a mesma para Windows e Android.
