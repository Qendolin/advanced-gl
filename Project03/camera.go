package main

import "github.com/go-gl/mathgl/mgl32"

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
	t := mgl32.Translate3D(-cam.Position[0], -cam.Position[1], -cam.Position[2])
	cam.ViewMatrix = r.Mat4().Mul4(t)
}

func (cam *Camera) UpdateProjectionMatrix() {
	w, h := cam.ViewportDimension[0], cam.ViewportDimension[1]
	n, f := cam.ClippingPlanes[0], cam.ClippingPlanes[1]
	cam.ProjectionMatrix = mgl32.Perspective(70, w/h, n, f)
}

func (cam *Camera) Quaternion() mgl32.Quat {
	return mgl32.AnglesToQuat(cam.Orientation[0]*Deg2Rad, cam.Orientation[1]*Deg2Rad, cam.Orientation[2]*Deg2Rad, mgl32.XYZ)
}

func (cam *Camera) Fly(vec mgl32.Vec3) {
	r := cam.Quaternion()
	cam.Position = cam.Position.Add(r.Conjugate().Rotate(vec))
}
