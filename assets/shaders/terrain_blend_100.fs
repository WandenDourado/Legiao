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
float side(float uv, float linked) { return mix(smoothstep(0.01, edgeWidth, uv), 1.0, linked); }
void main() {
    vec2 uv = (fragTexCoord - tileRect.xy) / tileRect.zw;
    float mask = min(min(side(uv.y, edge.x), side(1.0-uv.x, edge.y)), min(side(1.0-uv.y, edge.z), side(uv.x, edge.w)));
    if (edge.x > 0.5 && edge.w > 0.5 && corner.x < 0.5) mask = min(mask, smoothstep(0.02, edgeWidth, length(uv)));
    if (edge.x > 0.5 && edge.y > 0.5 && corner.y < 0.5) mask = min(mask, smoothstep(0.02, edgeWidth, length(uv-vec2(1.0,0.0))));
    if (edge.z > 0.5 && edge.y > 0.5 && corner.z < 0.5) mask = min(mask, smoothstep(0.02, edgeWidth, length(uv-vec2(1.0,1.0))));
    if (edge.z > 0.5 && edge.w > 0.5 && corner.w < 0.5) mask = min(mask, smoothstep(0.02, edgeWidth, length(uv-vec2(0.0,1.0))));
    float n = fract(sin(dot(uv*127.0, vec2(12.9898,78.233)))*43758.5453);
    mask = clamp(mask + (n-0.5)*0.16, 0.0, 1.0);
    vec4 src = texture2D(texture0, fragTexCoord);
    gl_FragColor = vec4(src.rgb, src.a*mask)*fragColor*colDiffuse;
}
