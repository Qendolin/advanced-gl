package libscn

import (
	"advanced-gl/Project03/libutil"

	"github.com/go-gl/mathgl/mgl32"
)

type Camera struct {
	Position mgl32.Vec3
	// pitch, yaw, roll in degrees
	Orientation mgl32.Vec3
	// in degrees
	VerticalFov       float32
	ViewportDimension mgl32.Vec2
	ClippingPlanes    mgl32.Vec2
	ViewMatrix        mgl32.Mat4
	ProjectionMatrix  mgl32.Mat4
}

func (cam *Camera) UpdateViewMatrix() {
	r := cam.Quaternion()
	t := mgl32.Translate3D(cam.Position[0], cam.Position[1], cam.Position[2])
	// The view matrix is (by definition) the inverse of the camera matrix.
	// This is the "correct" (but not the only) way to calculate it.
	// Don't forget: positive yaw angle represents a CCW rotation around the Y axis
	// and a positve pitch angle is a CCW rotation around the X axis
	cam.ViewMatrix = t.Mul4(r.Mat4()).Inv()
}

func (cam *Camera) UpdateProjectionMatrix() {
	w, h := cam.ViewportDimension[0], cam.ViewportDimension[1]
	n, f := cam.ClippingPlanes[0], cam.ClippingPlanes[1]
	cam.ProjectionMatrix = mgl32.Perspective(mgl32.DegToRad(cam.VerticalFov), w/h, n, f)
}

func (cam *Camera) Quaternion() mgl32.Quat {
	// Note the rotation order is Z, Y, X
	return mgl32.AnglesToQuat(cam.Orientation[2]*libutil.Deg2Rad, cam.Orientation[1]*libutil.Deg2Rad, cam.Orientation[0]*libutil.Deg2Rad, mgl32.ZYX)
}

func (cam *Camera) Fly(vec mgl32.Vec3) {
	r := cam.Quaternion()
	cam.Position = cam.Position.Add(r.Rotate(vec))
}
