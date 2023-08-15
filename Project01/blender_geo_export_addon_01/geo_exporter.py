import struct  # https://docs.python.org/3/library/struct.html


def write_geometry(context, obs, transform, file):
    meshes = set([ob.data for ob in obs if ob.type == 'MESH'])
    count = len(meshes)
    context.window_manager.progress_begin(0, count)
    file.write(struct.pack("<I", 0xc9dae18c))
    file.write(struct.pack("<I", count))
    offsets_offset = file.tell()
    file.seek(count * 4, 1)
    offsets = []
    progress = 0
    for me in meshes:
        print(f"Writing '{me.name}' [{progress+1}/{count}]...")
        offsets.append(file.tell())

        name = me.name.encode('utf-8')
        vertices, indices = _convert_mesh(me, transform)

        print(f"{len(vertices)} vertices, {len(indices)} indices")

        file.write(struct.pack("<I", 0xc9dae18c))
        file.write(struct.pack("<I", len(name)))
        file.write(struct.pack('<I', len(vertices)))
        file.write(struct.pack('<I', len(indices)))

        file.write(name)
        for (pos, norm, uv) in vertices:
            file.write(struct.pack('<3f', pos[0], pos[1], pos[2]))
            file.write(struct.pack(
                '<3f', norm[0], norm[1], norm[2]))
            file.write(struct.pack('<2f', uv[0], uv[1]))

        short = len(indices) < 0xffff
        for i in indices:
            file.write(struct.pack('<H' if short else '<I', i))
        if short and len(indices) % 2 == 1:
            file.write(struct.pack('<H', 0))
        progress += 1
        context.window_manager.progress_update(progress)

    file.seek(offsets_offset)
    for o in offsets:
        file.write(struct.pack('<I', o))
    context.window_manager.progress_end()


def _convert_mesh(me, transform):
    me = me.copy()

    me.calc_loop_triangles()
    me.calc_normals_split()

    me.transform(transform)

    vertices = dict()
    indices = []
    index = 0

    uvmap = me.uv_layers.active
    loops = me.loop_triangles

    if len(loops) == 0:
        print("Mesh has no triangulated loops")

    for tri in loops:
        for i in range(3):
            vert_index = tri.vertices[i]
            loop_index = tri.loops[i]

            pos = tuple(me.vertices[vert_index].co)
            norm = tuple(me.loops[loop_index].normal)
            uv = tuple(uvmap.data[loop_index].uv)
            vert = (pos, norm, uv)

            if vert in vertices:
                indices.append(vertices[vert])
            else:
                vertices[vert] = index
                indices.append(index)
                index += 1

    vertices = list(vertices.keys())
    return (vertices, indices)
