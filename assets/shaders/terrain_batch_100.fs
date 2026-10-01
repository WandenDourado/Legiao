#version 100
// Versao GLSL ES 2.0 do terrain_batch_330.fs. Ver o comentario de la para o
// porque de a mascara vir de textura e o indice da celula vir da cor.
precision mediump float;
varying vec2 fragTexCoord;
varying vec4 fragColor;

uniform sampler2D texture0;   // a textura do material
uniform sampler2D maskTex;    // W x H, um texel por celula: R = bordas, G = cantos
uniform vec4 colDiffuse;

uniform vec2 gridSize;        // largura e altura do mapa, em celulas
uniform float edgeWidth;      // a rampa desta PILHA (nao global)
uniform float spanF;          // quantas celulas a folha cobre num lado

// bitAt sem exp2/mod inteiro: ES 2.0 tem os dois, mas a precisao mediump nao
// aguenta 255 dividido por 128 com seguranca, entao a divisao e feita por
// constante literal e o floor faz o resto.
float bitAt(float packed, float weight) {
    float v = floor(packed * 255.0 + 0.5);
    return floor(mod(v / weight, 2.0));
}

// edgeWidth pode ser 0.01 (a borda dura de proposito da pilha do castelo).
// smoothstep(a,b,x) e INDEFINIDO com a >= b; no mobile isso vira NaN e apaga a
// celula inteira. Os limites sao derivados de edgeWidth para que uma pilha de
// borda dura continue dura, e nao para que fique macia.
float rampLo(float w) { return min(0.01, w * 0.5); }
float rampHi(float w) { return max(w, rampLo(w) + 0.002); }
float cornerLo(float w) { return min(0.02, w * 0.5); }
float cornerHi(float w) { return max(w, cornerLo(w) + 0.002); }
float side(float uv, float linked) {
    return mix(smoothstep(rampLo(edgeWidth), rampHi(edgeWidth), uv), 1.0, linked);
}

void main() {
    vec2 cell = floor(fragColor.rg * 255.0 + 0.5);

    vec2 window = vec2(mod(cell.x, spanF), mod(cell.y, spanF));
    vec2 uv = clamp(fragTexCoord * spanF - window, 0.0, 1.0);

    vec4 packed = texture2D(maskTex, (cell + 0.5) / gridSize);

    // ATALHO DA CELULA CERCADA — ver terrain_batch_330.fs para o porque. Aqui
    // ele vale ainda mais: e o mobile que paga o preco do empilhamento, e o
    // `sin()` do dither e caro nas GPUs de bloco. O teste usa o mesmo
    // floor(x*255+0.5) que bitAt ja usa, entao nao introduz suposicao de
    // precisao nova em mediump.
    if (floor(packed.r * 255.0 + 0.5) > 14.5 && floor(packed.g * 255.0 + 0.5) > 14.5) {
        gl_FragColor = texture2D(texture0, fragTexCoord) * colDiffuse;
        return;
    }

    vec4 edge = vec4(bitAt(packed.r, 1.0), bitAt(packed.r, 2.0),
                     bitAt(packed.r, 4.0), bitAt(packed.r, 8.0));
    vec4 corner = vec4(bitAt(packed.g, 1.0), bitAt(packed.g, 2.0),
                       bitAt(packed.g, 4.0), bitAt(packed.g, 8.0));

    float mask = min(min(side(uv.y, edge.x), side(1.0-uv.x, edge.y)),
                     min(side(1.0-uv.y, edge.z), side(uv.x, edge.w)));
    if (edge.x > 0.5 && edge.w > 0.5 && corner.x < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv)));
    if (edge.x > 0.5 && edge.y > 0.5 && corner.y < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(1.0,0.0))));
    if (edge.z > 0.5 && edge.y > 0.5 && corner.z < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(1.0,1.0))));
    if (edge.z > 0.5 && edge.w > 0.5 && corner.w < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(0.0,1.0))));

    vec2 texel = floor(uv * 128.0);
    float n = fract(sin(dot(texel, vec2(12.9898,78.233)))*12758.5453);
    float ramp = mask * (1.0 - mask) * 4.0;
    mask = clamp(mask + (n-0.5)*0.16*ramp, 0.0, 1.0);

    vec4 src = texture2D(texture0, fragTexCoord);
    gl_FragColor = vec4(src.rgb, src.a*mask)*colDiffuse;
}
