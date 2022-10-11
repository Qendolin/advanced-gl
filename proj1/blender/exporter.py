import struct  # https://docs.python.org/3/library/struct.html
import pathlib
import bpy
import json
from mathutils import (Matrix, Quaternion, Vector)
from bpy_extras.io_utils import (
    axis_conversion,
    orientation_helper
)
bl_info = {
    "name": "Geo Format Exporter",
    "description": "Writes geometry format to disk, based on exporter by Mark Kughler",
    "author": "Qendolin",
    "version": (1, 0),
    "blender": (3, 2, 1),
    "location": "File > Export > GEO",
    "warning": "",
    "wiki_url": "",
    "tracker_url": "",
    "support": 'COMMUNITY',
    "category": "Import-Export"
}


def convert_mesh(me, transform):
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


def convert_object(me, transform):
    me = me.copy().to_mesh()

    me.calc_loop_triangles()
    me.calc_normals_split()

    me.transform(transform)

    return me


def connects_to_socket(node, target, socket):
    for out in node.outputs:
        for link in out.links:
            if link.to_node == target:
                return link.to_socket == socket
            elif connects_to_socket(link.to_node, target, socket):
                return True
    return False


def extract_images(tree, target, sockets):
    imgs = [node for node in tree.nodes if isinstance(
        node, bpy.types.ShaderNodeTexImage)]
    results = []
    for sock in sockets:
        found = None
        for img in imgs:
            if connects_to_socket(img, target, sock):
                found = img
                break
        results.append(found)
        if found is not None:
            imgs.remove(found)

    return results


@orientation_helper(axis_forward='-Z', axis_up='Y')
class ObjectExport(bpy.types.Operator):
    """My object export script"""
    bl_idname = "object.export_geo"
    bl_label = "Geo Format Export"
    bl_options = {'REGISTER', 'UNDO'}
    filename_ext = ".geo"

    write_geometry: bpy.props.BoolProperty(
        name="Write .geo File",
        description="",
        default=True,
    )
    write_scene: bpy.props.BoolProperty(
        name="Write .scn File",
        description="",
        default=True,
    )
    filter_glob: bpy.props.StringProperty(
        default="*.geo", options={'HIDDEN'}, maxlen=255)
    global_scale: bpy.props.FloatProperty(
        name="Scale",
        min=0.01, max=1000.0,
        default=1.0,
    )

    filepath: bpy.props.StringProperty(subtype='FILE_PATH')

    def execute(self, context):
        if(context.active_object is not None and context.active_object.mode == 'EDIT'):
            bpy.ops.object.mode_set(mode='OBJECT')

        deps = context.evaluated_depsgraph_get()
        axis_transform = axis_conversion(
            to_forward=self.axis_forward,
            to_up=self.axis_up,
        ).to_4x4()
        scale_transform = Matrix([[abs(v) for v in row]
                                 for row in axis_transform])
        transform = (
            Matrix.Scale(self.global_scale, 4) @
            axis_transform
        )

        if self.filepath.lower().endswith(".geo"):
            self.filepath = self.filepath[:-4]

        if self.write_geometry:
            with open(self.filepath + ".geo", 'wb') as file:
                all_obs = [
                    ob.evaluated_get(deps) for ob in context.selected_objects if ob.type == 'MESH']
                meshes = set([ob.data for ob in all_obs])
                count = len(meshes)
                context.window_manager.progress_begin(0, count)
                file.write(struct.pack("<I", count))
                file.seek(count * 4, 1)
                offsets = []
                progress = 0
                for me in meshes:
                    print(f"Writing '{me.name}'...")
                    offsets.append(file.tell())

                    name = me.name.encode('utf-8')
                    vertices, indices = convert_mesh(me, transform)

                    print(f"{len(vertices)} vertices, {len(indices)} indices")

                    file.write(struct.pack(">I", 0xdeadbeef))
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

                file.seek(4)
                for o in offsets:
                    file.write(struct.pack('<I', o))
                context.window_manager.progress_end()

        if self.write_scene:
            with open(self.filepath + ".scn", 'w') as file:
                obs = context.selected_objects
                out_objects = []
                out_lights = []
                materials = set()
                for ob in obs:
                    ob = ob.evaluated_get(deps)
                    mat = None
                    if len(ob.material_slots) > 0:
                        mat = ob.material_slots[0].material
                        materials.add(mat)
                    pos = list(transform @ ob.location)
                    scale = list(scale_transform @ ob.scale)
                    rot = ob.rotation_euler.to_quaternion()
                    if ob.rotation_mode == "QUATERNION":
                        rot = ob.rotation_quaternion
                    elif ob.rotation_mode == "AXIS_ANGLE":
                        rot = ob.rotation_axis_angle.to_quaternion()
                    rotAxis = scale_transform.copy(
                    ) @ Vector([rot.x, rot.y, rot.z])
                    rot = list(Quaternion(
                        [rot.w, rotAxis.x, rotAxis.y, rotAxis.z]))
                    if ob.type == "MESH":
                        out_objects.append({
                            "name": ob.name,
                            "mesh": ob.data.name,
                            "position": pos,
                            "rotation": rot,
                            "scale": scale,
                            "material": mat.name if mat is not None else ""
                        })
                    elif ob.type == "LIGHT":
                        light = ob.data
                        data = {
                            "name": ob.name,
                            "type": light.type,
                            "position": pos,
                            "color": list(light.color),
                            "radius": light.cutoff_distance if light.use_custom_distance else 0,
                            "attenuation": 2 / (light.distance ** 2),
                            "shadow": light.use_shadow,
                        }
                        if light.type == "SPOT":
                            data["outerAngle"] = light.spot_size
                            data["innerAngle"] = light.spot_size * \
                                (1 - light.spot_blend)
                            data["rotation"] = rot
                            data["power"] = light.energy
                            data["attenuation"] = light.quadratic_coefficient
                        elif light.type == "POINT":
                            data["power"] = light.energy
                            data["attenuation"] = light.quadratic_coefficient
                        elif light.type == "SUN":
                            data["power"] = light.energy
                            data["rotation"] = rot

                        out_lights.append(data)
                    else:
                        print(
                            f"Cannot export object {ob.name}, unsupported type")

                out_materials = []
                for mat in materials:
                    if not mat.use_nodes:
                        print(
                            f"Cannot export material {mat.name}, does not use nodes")
                        continue
                    nodes = mat.node_tree
                    out = nodes.get_output_node("ALL")
                    if out.bl_idname != "ShaderNodeOutputMaterial":
                        print(
                            f"Cannot export material {mat.name}, output node is not ShaderNodeOutputMaterial")
                        continue
                    links = out.inputs[0].links
                    if len(links) != 1:
                        continue
                    shader = links[0].from_node
                    out_material = {
                        "name": mat.name,
                    }
                    if shader.bl_idname == "ShaderNodeBsdfPrincipled":
                        [texImg, normImg] = extract_images(
                            mat.node_tree, shader, [shader.inputs[0], shader.inputs[22]])
                        if texImg is not None:
                            out_material["albedoTexture"] = pathlib.PurePath(
                                texImg.image.filepath).stem
                        if normImg is not None:
                            out_material["normalTexture"] = pathlib.PurePath(
                                normImg.image.filepath).stem
                    else:
                        print(
                            f"Cannot export material {mat.name}, unsupported shader type")
                        continue
                    out_materials.append(out_material)

                json.dump({"lights": out_lights, "objects": out_objects, "materials": out_materials},
                          file, indent=4)
        print("finished")
        return {'FINISHED'}

    def invoke(self, context, event):
        context.window_manager.fileselect_add(self)
        return {'RUNNING_MODAL'}


# Add trigger into a dynamic menu
def menu_func_export(self, context):
    self.layout.operator(ObjectExport.bl_idname, text="Geometry Export (.geo)")


def register():
    bpy.utils.register_class(ObjectExport)
    bpy.types.TOPBAR_MT_file_export.append(menu_func_export)


def unregister():
    bpy.utils.unregister_class(ObjectExport)
    bpy.types.TOPBAR_MT_file_export.remove(menu_func_export)


if __name__ == "__main__":
    register()
