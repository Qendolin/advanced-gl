import bpy
from bpy_extras.image_utils import load_image

def conv_material(mat):
	if not mat.use_nodes:
		raise Exception(f"material does not use nodes")
	nodes = mat.node_tree
	out = nodes.get_output_node("ALL")
	if out.bl_idname != "ShaderNodeOutputMaterial":
		raise Exception(f"unsupported output node type '{out.bl_idname}'")

	links = out.inputs["Surface"].links
	if len(links) != 1:
		raise Exception("output node has no connected shader")

	out_material = {
		"name": mat.name,
	}

	shader = links[0].from_node
	textures = _conv_shader(shader)
	for tex_name in textures:
		if(textures[tex_name] is None):
			continue
		out_material[tex_name] = pathlib.PurePath(textures[tex_name].filepath).stem

	return out_material

def _conv_shader(shader):
	if shader.bl_idname == "ShaderNodeBsdfPrincipled":
		return _conv_principled(shader)
	else:
		raise Exception(f"unsupported shader type '{shader.bl_idname}'")

def _conv_principled(shader):
	[albedo, normal, roughness, metalness, emission] = _extract_images(mat.node_tree, shader, [
		shader.inputs["Base Color"], 
		shader.inputs["Normal"],
		shader.inputs["Roughness"],
		shader.inputs["Metallic"]])
	
	return {
		albedoTexture: albedo.image if albedo else None,
		normalTexture: normal if normal else None,
		ormTexture: _pack_orm_texture(None, roughness, metalness)
	}

def _pack_orm_texture(o, r, m):
	o_img = o.image if o else None
	r_img = r.image if r else None
	m_img = m.image if m else None

	combined_image = bpy.data.images.new(name="CombinedORMTexture.000", width=r_img.size[0], height=r_img.size[1])

	# https://blender.stackexchange.com/a/3678
	combined_pixels = list(combined_image.pixels)

	# TODO: generalize default values, texture and constant values
	if o_img:
		combined_pixels[0::4] = o_img.pixels[:][0::4]
	else:
		combined_pixels[0::4] = [1.0] * (len(combined_pixels) // 4)

	if r_img:
		combined_pixels[1::4] = r_img.pixels[:][1::4]
	else:
		combined_pixels[1::4] = [0.75] * (len(combined_pixels) // 4)

	if m_img:
		combined_pixels[2::4] = m_img.pixels[:][2::4]
	
	combined_pixels[3::4] = [1.0] * (len(combined_pixels) // 4)

	combined_image.pixels[:] = combined_pixels
	
	# Save the combined image to disk
	combined_image.filepath_raw = "//combined_texture.png"
	combined_image.file_format = 'PNG'
	combined_image.save()


def _connects_to_socket(node, target, socket):
    for out in node.outputs:
        for link in out.links:
            if link.to_node == target:
                return link.to_socket == socket
            elif _connects_to_socket(link.to_node, target, socket):
                return True
    return False


def _extract_image(tree, target, sockets):
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