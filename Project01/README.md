This is my first deferred rendering project.

## Running the Demo

System Requirements:
- Opengl `4.5` or higher
- Go 1.18 or higher
- A 1920x1080 display or higher  

Use `go run .` to start the demo.   
**The first compilation might take a few minutes!!**

You can also download one of the prebuilt binaries from GitHub, just make sure to place them next to the `assets` folder.

There are two launch options:

- `-disable-shader-cache`: To disable the usage of the `.shadercache` folder.
- `-enable-compatibility-profile`: To create an OpenGL compatability context

### Issues

For some reason I only get a single frame when running the demo on my Intel UHD Graphics.

## Controls

Press <kbd>Alt</kbd> to lock/unlock the cursor.

Use <kbd>W</kbd>, <kbd>A</kbd>, <kbd>S</kbd>, <kbd>D</kbd> to move horizontally, <kbd>Space</kbd> and <kbd>Ctrl</kbd> to move vertically. 
Hold <kbd>Shift</kbd> to fly faster.

Press <kbd>Strg</kbd>+<kbd>F11</kbd> to dump the framebuffers into `./dump`.

## Tech

The demo uses 'regular' deferred rendering and supports

- Physically Based Bloom
- Screen Space Ambient Occlusion
- Percentage Closer Filtered Shadow Mapping for directinal lights
- Multicolored directional, spot and point lights
- Screen Dithering
- Color Mapping
- ImGui