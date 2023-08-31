import pathlib
import bpy
from mathutils import Matrix
from bpy_extras.io_utils import (
    axis_conversion,
    orientation_helper
)
import os.path
import re

from .geo_exporter import write_geometry
from .scn_exporter import write_scene

bl_info = {
    "name": "Geo Format Exporter 03",
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

# Add trigger into a dynamic menu


def menu_func_export(self, context):
    self.layout.operator(ObjectExport.bl_idname, text="Geometry Export 03 (.geo)")


@orientation_helper(axis_forward='-Z', axis_up='Y')
class ObjectExport(bpy.types.Operator):
    """My object export script"""
    bl_idname = "object.export_geo_03"
    bl_label = "Geo Format Export 03"
    bl_options = {'REGISTER', 'UNDO'}

    write_geometry: bpy.props.BoolProperty(
        name="Write .geo Files",
        description="",
        default=True,
    )
    write_scene: bpy.props.BoolProperty(
        name="Write .scn Files",
        description="",
        default=False,
    )
    global_scale: bpy.props.FloatProperty(
        name="Scale",
        min=0.01, max=1000.0,
        default=1.0,
    )

    directory: bpy.props.StringProperty(subtype='DIR_PATH')

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

        obs = [ob.evaluated_get(deps)
               for ob in context.selected_objects if not ob.hide_render]
        obs.sort(key=lambda ob: ob.name)

        if self.write_geometry:
            progress = 0
            mesh_obs = set([ob for ob in obs if ob.type == 'MESH'])
            count = len(mesh_obs)
            context.window_manager.progress_begin(0, count)
            for ob in mesh_obs:
                me = ob.data
                print(f"Writing '{ob.name}' [{progress+1}/{count}]...")
                filename = re.sub(r'[^\w\s-]', '', ob.name.lower())
                filename = re.sub(r'[-\s]+', '-', filename).strip('-_')
                path = os.path.join(self.directory, filename + ".geo")
                with open(path, 'wb') as file:
                    write_geometry(me, transform, file)
                context.window_manager.progress_update(progress)

            context.window_manager.progress_end()

        if self.write_scene:
            with open(os.path.join(self.directory, "scene.scn"), 'w') as file:
                write_scene(context, obs, scale_transform, transform, file)

        print("finished")
        return {'FINISHED'}

    def invoke(self, context, event):
        context.window_manager.fileselect_add(self)
        return {'RUNNING_MODAL'}


def register():
    bpy.utils.register_class(ObjectExport)
    bpy.types.TOPBAR_MT_file_export.append(menu_func_export)


def unregister():
    bpy.utils.unregister_class(ObjectExport)
    bpy.types.TOPBAR_MT_file_export.remove(menu_func_export)


if __name__ == "__main__":
    register()
