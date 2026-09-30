#version 100
// Fades one terrain material out at the borders it does not share with a
// neighbour of the same or higher priority. The renderer draws the material
// itself as texture0 over the passes already on screen, so the output is an
// alpha ramp instead of a mix against a fixed grass base — that is what makes
// a stone yard dissolve INTO the dirt path it touches.
precision mediump float;
varying vec2 fragTexCoord;
varying vec4 fragColor;
uniform sampler2D texture0;
uniform vec4 edge;
uniform vec4 corner;
uniform float edgeWidth;
uniform vec4 colDiffuse;
// tileRect e a janela da textura que ESTA celula usa: xy = canto, zw = tamanho,
// em UV. Existe porque uma textura pode cobrir NxN celulas (pixels 1:1 em vez
// de espremer a folha inteira em 128px), e nesse caso fragTexCoord percorre so
// a janela — nao 0..1. A mascara de borda precisa de 0..1 DENTRO da celula, a
// amostragem precisa da coordenada real da textura; sem separar as duas, mudar
// a janela apagava o desbotamento de borda. Com uma textura por celula,
// tileRect e (0,0,1,1) e local == fragTexCoord.
uniform vec4 tileRect;
// edgeWidth pode ser 0.01 (a borda dura de proposito da pilha do castelo).
// smoothstep(a,b,x) e INDEFINIDO com a >= b; no mobile isso vira NaN e apaga a
// celula inteira. Os limites sao derivados de edgeWidth para que uma pilha de
// borda dura continue dura, e nao para que fique macia.
float rampLo(float w) { return min(0.01, w * 0.5); }
float rampHi(float w) { return max(w, rampLo(w) + 0.002); }
float cornerLo(float w) { return min(0.02, w * 0.5); }
float cornerHi(float w) { return max(w, cornerLo(w) + 0.002); }
float side(float uv, float linked) { return mix(smoothstep(rampLo(edgeWidth), rampHi(edgeWidth), uv), 1.0, linked); }
void main() {
    // ATALHO DA CELULA CERCADA — ver terrain_blend_330.fs.
    if (edge.x > 0.5 && edge.y > 0.5 && edge.z > 0.5 && edge.w > 0.5 &&
        corner.x > 0.5 && corner.y > 0.5 && corner.z > 0.5 && corner.w > 0.5) {
        gl_FragColor = texture2D(texture0, fragTexCoord)*fragColor*colDiffuse;
        return;
    }
    vec2 uv = (fragTexCoord - tileRect.xy) / tileRect.zw;
    float mask = min(min(side(uv.y, edge.x), side(1.0-uv.x, edge.y)), min(side(1.0-uv.y, edge.z), side(uv.x, edge.w)));
    if (edge.x > 0.5 && edge.w > 0.5 && corner.x < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv)));
    if (edge.x > 0.5 && edge.y > 0.5 && corner.y < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(1.0,0.0))));
    if (edge.z > 0.5 && edge.y > 0.5 && corner.z < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(1.0,1.0))));
    if (edge.z > 0.5 && edge.w > 0.5 && corner.w < 0.5) mask = min(mask, smoothstep(cornerLo(edgeWidth), cornerHi(edgeWidth), length(uv-vec2(0.0,1.0))));
    // O dither e ANCORADO NO TEXEL e PESADO PELA RAMPA, exatamente como em
    // terrain_batch_*.fs. Eram as duas diferencas que sobravam entre os dois
    // caminhos: sem o floor o ruido fervia com a camera, e sem a rampa ele
    // furava o INTERIOR da celula com ate 8% de alfa — o que deixava a camada
    // de baixo aparecer onde nada deveria atravessar, e teria feito a oclusao
    // parecer um defeito de desenho neste caminho.
    float n = fract(sin(dot(floor(uv*128.0), vec2(12.9898,78.233)))*12758.5453);
    float ramp = mask * (1.0 - mask) * 4.0;
    mask = clamp(mask + (n-0.5)*0.16*ramp, 0.0, 1.0);
    vec4 src = texture2D(texture0, fragTexCoord);
    gl_FragColor = vec4(src.rgb, src.a*mask)*fragColor*colDiffuse;
}
