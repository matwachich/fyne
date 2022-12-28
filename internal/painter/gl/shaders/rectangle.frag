#version 110

uniform vec4 rect_coords; //x1 [0], x2 [1], y1 [2], y2 [3]; coords of the rect_frame
uniform float stroke;
uniform float radius;
uniform vec4 fill_color;
uniform vec4 stroke_color;
varying vec4 frame_resolution; //size of view/window = x,y; z = pixScale (w not used); 


float RectSDF(vec2 p, vec2 b, float r)
{
    vec2 d = abs(p) - b + vec2(r);
	return min(max(d.x, d.y), 0.0) + length(max(d, 0.0)) - r;   
}

void main() {

    if ( radius == 0.0 ){
        vec4 color = fill_color;
        
        if (gl_FragCoord.x >= rect_coords[1] - stroke ){
            color = stroke_color;
        } else if (gl_FragCoord.x <= rect_coords[0] + stroke){
            color = stroke_color;
        } else if (gl_FragCoord.y <= frame_resolution.y - rect_coords[3] + stroke ){
            color = stroke_color;
        } else if (gl_FragCoord.y >= frame_resolution.y - rect_coords[2] - stroke ){
            color = stroke_color;
        }

        gl_FragColor = color;

    } else {
        float pixScale = frame_resolution.z;  // -> for better readability
        vec4 norm_coords = vec4(rect_coords[0] * pixScale, rect_coords[1] * pixScale, (frame_resolution.y - rect_coords[3]) * pixScale, (frame_resolution.y - rect_coords[2]) * pixScale);
        float u_fHalfBorderThickness = stroke / 2.0;
        vec2 u_v2HalfShapeSizePx =  vec2(norm_coords[1] - norm_coords[0], norm_coords[3] - norm_coords[2]) / 2.0 - vec2(u_fHalfBorderThickness);
        vec2 v_v2CenteredPos = (gl_FragCoord.xy - vec2(norm_coords[0] + norm_coords[1], norm_coords[2] + norm_coords[3]) / 2.0);

        float fDist = RectSDF(v_v2CenteredPos, u_v2HalfShapeSizePx, radius - u_fHalfBorderThickness);
    
        vec4 v4FromColor = stroke_color; //Always the border color. If no border, this still should be set
        vec4 v4ToColor = vec4(0.5, 0.5, 0.5, 0.5); //Outside color
        if ( frame_resolution.x != 800.0 ){
            v4ToColor = vec4(1.0, 0.0, 0.0, 0.5); //Outside color
        } else if ( frame_resolution.y != 600.0 ){
            v4ToColor = vec4(1.0, 0.0, 0.0, 0.5); //Outside color
        } 
    
        if (u_fHalfBorderThickness > 0.0)
        {
            if (fDist < 0.0)
            {
                v4ToColor = fill_color;   
            } 
            
            fDist = abs(fDist) - u_fHalfBorderThickness;
        }
    
        float fBlendAmount = smoothstep(-1.0, 1.0, fDist);
    
        // final color
        gl_FragColor = mix(v4FromColor, v4ToColor, fBlendAmount);
        //gl_FragColor = vec4(vec3(fBlendAmount), 1.0);
        //gl_FragColor = vec4(vec3(abs(dist) / (2.0 * corner)), 1.0);
    }

}
