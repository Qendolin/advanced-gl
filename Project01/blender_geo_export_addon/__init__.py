import pathlib
import bpy
from mathutils import Matrix
from bpy_extras.io_utils import (
    axis_conversion,
    orientation_helper
)
import io
import re

from .geo_exporter import write_geometry
from .scn_exporter import write_scene

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

# Add trigger into a dynamic menu


def menu_func_export(self, context):
    self.layout.operator(ObjectExport.bl_idname, text="Geometry Export (.geo)")


def install_lz4():
    try:
        import lz4
    except ImportError:
        import subprocess
        import sys
        import os

        print("Installing lz4...")

        # Reference: https://devtalk.blender.org/t/can-3rd-party-modules-ex-scipy-be-installed-when-an-add-on-is-installed/9709/11
        py_exec = str(sys.executable)
        # Get lib directory
        lib = os.path.join(pathlib.Path(py_exec).parent.parent, "lib")
        # Ensure pip is installed
        print("Checking for pip")
        subprocess.call([py_exec, "-m", "ensurepip", "--user"])
        # Update pip (not mandatory)
        print("Updating pip")
        subprocess.call([py_exec, "-m", "pip", "install", "--upgrade", "pip"])
        # Install package
        print("Installing lz4")
        subprocess.call([py_exec, "-m", "pip", "install", f"--target={str(lib)}", "lz4"])

        import lz4


@orientation_helper(axis_forward='-Z', axis_up='Y')
class ObjectExport(bpy.types.Operator):
    """My object export script"""
    bl_idname = "object.export_geo"
    bl_label = "Geo Format Export"
    bl_options = {'REGISTER', 'UNDO'}

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
        default="*.geo;*.geo.lz4;*.scn;*.scn.lz4", options={'HIDDEN'}, maxlen=255)
    global_scale: bpy.props.FloatProperty(
        name="Scale",
        min=0.01, max=1000.0,
        default=1.0,
    )
    compression: bpy.props.BoolProperty(
        name="Compress files",
        description="",
        default=True,
    )
    compression_level: bpy.props.IntProperty(
        name="Compression level",
        description="",
        max=16,
        min=0,
        default=8,
    )

    filepath: bpy.props.StringProperty(subtype='FILE_PATH')

    def execute(self, context):
        import lz4.frame

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

        self.filepath = re.sub('\\.(geo|scn)(\\.lz4)?$', '', self.filepath, flags=re.IGNORECASE)

        obs = [ob.evaluated_get(deps)
               for ob in context.selected_objects if not ob.hide_render]
        obs.sort(key=lambda ob: ob.name)

        if self.write_geometry:
            if self.compression:
                with open(self.filepath + ".geo.lz4", 'wb') as file:
                    data = io.BytesIO()
                    write_geometry(context, obs, transform, data)
                    print("Compressing...")
                    compressed = lz4.frame.compress(data.getvalue(), compression_level=self.compression_level)
                    file.write(compressed)
            else:
                with open(self.filepath + ".geo", 'wb') as file:
                    write_geometry(context, obs, transform, file)

        if self.write_scene:
            if self.compression:
                with open(self.filepath + ".scn.lz4", 'wb') as file:
                    data = io.StringIO()
                    write_scene(context, obs, scale_transform, transform, data)
                    compressed = lz4.frame.compress(data.getvalue().encode(), compression_level=self.compression_level)
                    file.write(compressed)
            else:
                with open(self.filepath + ".scn", 'w') as file:
                    write_scene(context, obs, scale_transform, transform, file)

        print("finished")
        return {'FINISHED'}

    def invoke(self, context, event):
        context.window_manager.fileselect_add(self)
        return {'RUNNING_MODAL'}


def register():
    install_lz4()
    bpy.utils.register_class(ObjectExport)
    bpy.types.TOPBAR_MT_file_export.append(menu_func_export)


def unregister():
    bpy.utils.unregister_class(ObjectExport)
    bpy.types.TOPBAR_MT_file_export.remove(menu_func_export)


if __name__ == "__main__":
    register()
