
import pathlib
import bpy
import json
from mathutils import (Matrix, Quaternion, Vector)

def write_scene(context, obs, scale_transform, transform, file):
    out_objects = []
    out_lights = []
    materials = set()
    for ob in obs:
        mat = None
        if len(ob.material_slots) > 0:
            mat = ob.material_slots[0].material
            materials.add(mat)
        pos = [round(co, 4)
                for co in list(transform @ ob.location)]
        scale = list(scale_transform @ ob.scale)
        rot = ob.rotation_euler.to_quaternion()
        if ob.rotation_mode == "QUATERNION":
            rot = ob.rotation_quaternion
        elif ob.rotation_mode == "AXIS_ANGLE":
            rot = ob.rotation_axis_angle.to_quaternion()
        rotAxis = scale_transform.copy(
        ) @ Vector([rot.x, rot.y, rot.z])
        rot = [round(co, 4) for co in list(Quaternion(
            [rot.w, rotAxis.x, rotAxis.y, rotAxis.z]))]
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
            [texImg, normImg] = _extract_images(
                mat.node_tree, shader, [shader.inputs[0], shader.inputs[22]])
            if texImg is not None:
                out_material["albedoTexture"] = pathlib.PurePath(
                    texImg.image.filepath).stem
                out_material["transparent"] = texImg.image.alpha_mode != "NONE"
            if normImg is not None:
                out_material["normalTexture"] = pathlib.PurePath(
                    normImg.image.filepath).stem
        else:
            print(
                f"Cannot export material {mat.name}, unsupported shader type")
            continue
        out_materials.append(out_material)

    json.dump({"lights": out_lights, "objects": out_objects, "materials": out_materials},
                file, indent="\t")

def _connects_to_socket(node, target, socket):
    for out in node.outputs:
        for link in out.links:
            if link.to_node == target:
                return link.to_socket == socket
            elif _connects_to_socket(link.to_node, target, socket):
                return True
    return False


def _extract_images(tree, target, sockets):
    imgs = [node for node in tree.nodes if isinstance(
        node, bpy.types.ShaderNodeTexImage)]
    results = []
    for sock in sockets:
        found = None
        for img in imgs:
            if _connects_to_socket(img, target, sock):
                found = img
                break
        results.append(found)
        if found is not None:
            imgs.remove(found)

    return results