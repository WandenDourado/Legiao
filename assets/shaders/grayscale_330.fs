#version 330
// Draws a sprite with every colour removed but its transparency intact: this
// is what a dead player looks like. Luminance uses the usual perceptual
// weights, so a red robe and a blue one do not collapse to the same grey.
in vec2 fragTexCoord;
in vec4 fragColor;
out vec4 finalColor;
uniform sampler2D texture0;
uniform vec4 colDiffuse;
void main() {
    vec4 src = texture(texture0, fragTexCoord)*fragColor*colDiffuse;
    float lum = dot(src.rgb, vec3(0.299, 0.587, 0.114));
    finalColor = vec4(vec3(lum), src.a);
}
