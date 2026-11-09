//go:build android

package apprestart

/*
#cgo LDFLAGS: -landroid
#include <jni.h>
#include <stddef.h>
#include <unistd.h>
#include <android/native_activity.h>

// O comeco de `struct android_app` (android_native_app_glue.h, o mesmo
// arquivo desde o NDK r5): so precisamos do `activity`, o quarto campo. O
// header da glue nao esta no caminho de include deste pacote — ele vive
// dentro do modulo raylib-go.
struct legiao_app_head {
	void *userData;
	void *onAppCmd;
	void *onInputEvent;
	ANativeActivity *activity;
};

// Definida em rcore_android.c (pacote raylib-go). FRACA de proposito: o cgo
// liga cada pacote sozinho num teste (_cgo_.o) antes da ligacao final, e ali
// uma referencia forte a um simbolo de OUTRO pacote nao resolve. Na .so final
// ela aponta para a definicao do raylib.
extern struct legiao_app_head *GetAndroidApp(void) __attribute__((weak));

static int legiao_check(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

// legiao_restart pede ao Android a MESMA atividade de lancamento numa tarefa
// limpa (Intent.makeRestartActivityTask) e encerra este processo. So usa
// classes do sistema: FindClass numa thread nativa anexada ve o class loader
// do sistema, nao o do app.
//
// Devolve 0 so se nao voltar (o processo morre); negativo = nada feito.
static int legiao_restart(void) {
	if (GetAndroidApp == NULL) return -1;
	struct legiao_app_head *app = GetAndroidApp();
	if (app == NULL || app->activity == NULL) return -2;
	ANativeActivity *na = app->activity;
	JavaVM *vm = na->vm;
	JNIEnv *env = NULL;
	jint st = (*vm)->GetEnv(vm, (void **)&env, JNI_VERSION_1_6);
	if (st == JNI_EDETACHED) {
		if ((*vm)->AttachCurrentThread(vm, &env, NULL) != JNI_OK) return -3;
	} else if (st != JNI_OK) {
		return -3;
	}

	jobject act = na->clazz;
	jclass actCls = (*env)->GetObjectClass(env, act);
	jmethodID getPM = (*env)->GetMethodID(env, actCls, "getPackageManager", "()Landroid/content/pm/PackageManager;");
	jmethodID getPkg = (*env)->GetMethodID(env, actCls, "getPackageName", "()Ljava/lang/String;");
	jmethodID start = (*env)->GetMethodID(env, actCls, "startActivity", "(Landroid/content/Intent;)V");
	if (legiao_check(env) || !getPM || !getPkg || !start) return -4;

	jobject pm = (*env)->CallObjectMethod(env, act, getPM);
	jobject pkg = (*env)->CallObjectMethod(env, act, getPkg);
	if (legiao_check(env) || !pm || !pkg) return -5;

	jclass pmCls = (*env)->GetObjectClass(env, pm);
	jmethodID getLaunch = (*env)->GetMethodID(env, pmCls, "getLaunchIntentForPackage", "(Ljava/lang/String;)Landroid/content/Intent;");
	if (legiao_check(env) || !getLaunch) return -6;
	jobject launch = (*env)->CallObjectMethod(env, pm, getLaunch, pkg);
	if (legiao_check(env) || !launch) return -7;

	jclass intentCls = (*env)->FindClass(env, "android/content/Intent");
	if (legiao_check(env) || !intentCls) return -8;
	jmethodID getComp = (*env)->GetMethodID(env, intentCls, "getComponent", "()Landroid/content/ComponentName;");
	jmethodID mkRestart = (*env)->GetStaticMethodID(env, intentCls, "makeRestartActivityTask", "(Landroid/content/ComponentName;)Landroid/content/Intent;");
	if (legiao_check(env) || !getComp || !mkRestart) return -9;
	jobject comp = (*env)->CallObjectMethod(env, launch, getComp);
	if (legiao_check(env) || !comp) return -10;
	jobject restart = (*env)->CallStaticObjectMethod(env, intentCls, mkRestart, comp);
	if (legiao_check(env) || !restart) return -11;

	(*env)->CallVoidMethod(env, act, start, restart);
	if (legiao_check(env)) return -12;

	// O pedido ja esta com o sistema (startActivity e uma chamada sincrona
	// ao ActivityManager): matar o processo agora faz a atividade nova subir
	// num processo novo, com janela nova.
	_exit(0);
	return 0;
}
*/
import "C"

import "fmt"

// Supported: o celular reinicia sozinho.
const Supported = true

// Restart reabre o jogo e nao volta quando da certo. Quem chama tem de ter
// gravado as configuracoes ANTES (settings.Save). Um erro devolvido quer
// dizer que nada aconteceu — o jogo segue aberto e o jogador pode fechar e
// abrir na mao.
func Restart() error {
	if rc := C.legiao_restart(); rc != 0 {
		return fmt.Errorf("apprestart: reinicio falhou (codigo %d)", int(rc))
	}
	return nil
}
