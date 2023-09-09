package libgl

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
)

// var shaderMetaPattern = regexp.MustCompile(`(?m)^\/\/meta:(\w+)(.+)$`)
var shaderDefinePattern = regexp.MustCompile(`(?m)^\s*(\/\/)?\s*#define ([\w\d]+) ?(.*)$`)
var shaderVersionPattern = regexp.MustCompile(`(?m)^\s*#version.+$`)

type shaderPipeline struct {
	glId          uint32
	vertStage     ShaderProgram
	tessCtrlStage ShaderProgram
	tessEvalStage ShaderProgram
	geomStage     ShaderProgram
	fragStage     ShaderProgram
	compStage     ShaderProgram
}

type UnboundShaderPipeline interface {
	LabeledGlObject
	Bind() BoundShaderPipeline
	Attach(program ShaderProgram, stages int)
	ReAttach(stages int)
	Detach(stages int)
	Get(stage int) ShaderProgram
	FragmentStage() ShaderProgram
	VertexStage() ShaderProgram
	Id() uint32
	Delete()
}

type BoundShaderPipeline interface {
	UnboundShaderPipeline
}

func NewPipeline() UnboundShaderPipeline {
	var id uint32
	gl.CreateProgramPipelines(1, &id)
	return &shaderPipeline{
		glId: id,
	}
}

func (shaderPipeline *shaderPipeline) SetDebugLabel(label string) {
	setObjectLabel(gl.PROGRAM_PIPELINE, shaderPipeline.glId, label)
}

func (shaderPipeline *shaderPipeline) Attach(program ShaderProgram, stages int) {
	gl.UseProgramStages(shaderPipeline.glId, uint32(stages), program.Id())
	shaderPipeline.setStagesProgram(program, stages)
}

func (shaderPipeline *shaderPipeline) ReAttach(stages int) {
	if stages&gl.VERTEX_SHADER_BIT != 0 {
		gl.UseProgramStages(shaderPipeline.glId, gl.VERTEX_SHADER_BIT, shaderPipeline.vertStage.Id())
	}
	if stages&gl.TESS_CONTROL_SHADER_BIT != 0 {
		gl.UseProgramStages(shaderPipeline.glId, gl.TESS_CONTROL_SHADER_BIT, shaderPipeline.tessCtrlStage.Id())
	}
	if stages&gl.TESS_EVALUATION_SHADER_BIT != 0 {
		gl.UseProgramStages(shaderPipeline.glId, gl.TESS_EVALUATION_SHADER_BIT, shaderPipeline.tessEvalStage.Id())
	}
	if stages&gl.GEOMETRY_SHADER_BIT != 0 {
		gl.UseProgramStages(shaderPipeline.glId, gl.GEOMETRY_SHADER_BIT, shaderPipeline.geomStage.Id())
	}
	if stages&gl.FRAGMENT_SHADER_BIT != 0 {
		gl.UseProgramStages(shaderPipeline.glId, gl.FRAGMENT_SHADER_BIT, shaderPipeline.fragStage.Id())
	}
	if stages&gl.COMPUTE_SHADER_BIT != 0 {
		gl.UseProgramStages(shaderPipeline.glId, gl.COMPUTE_SHADER_BIT, shaderPipeline.compStage.Id())
	}
}

func (shaderPipeline *shaderPipeline) Detach(stages int) {
	gl.UseProgramStages(shaderPipeline.glId, uint32(stages), 0)
	shaderPipeline.setStagesProgram(nil, stages)
}

func (shaderPipeline *shaderPipeline) setStagesProgram(program ShaderProgram, stages int) {
	if stages&gl.VERTEX_SHADER_BIT != 0 {
		shaderPipeline.vertStage = program
	}
	if stages&gl.TESS_CONTROL_SHADER_BIT != 0 {
		shaderPipeline.tessCtrlStage = program
	}
	if stages&gl.TESS_EVALUATION_SHADER_BIT != 0 {
		shaderPipeline.tessEvalStage = program
	}
	if stages&gl.GEOMETRY_SHADER_BIT != 0 {
		shaderPipeline.geomStage = program
	}
	if stages&gl.FRAGMENT_SHADER_BIT != 0 {
		shaderPipeline.fragStage = program
	}
	if stages&gl.COMPUTE_SHADER_BIT != 0 {
		shaderPipeline.compStage = program
	}
}

func (shaderPipeline *shaderPipeline) Get(stage int) ShaderProgram {
	switch stage {
	case gl.VERTEX_SHADER:
		return shaderPipeline.vertStage
	case gl.TESS_CONTROL_SHADER:
		return shaderPipeline.tessCtrlStage
	case gl.TESS_EVALUATION_SHADER:
		return shaderPipeline.tessEvalStage
	case gl.GEOMETRY_SHADER:
		return shaderPipeline.geomStage
	case gl.FRAGMENT_SHADER:
		return shaderPipeline.fragStage
	case gl.COMPUTE_SHADER:
		return shaderPipeline.compStage
	}
	log.Panicf("%d is not a valid shader stage\n", stage)
	return nil
}

func (shaderPipeline *shaderPipeline) VertexStage() ShaderProgram {
	return shaderPipeline.vertStage
}

func (shaderPipeline *shaderPipeline) FragmentStage() ShaderProgram {
	return shaderPipeline.fragStage
}

func (shaderPipeline *shaderPipeline) Bind() BoundShaderPipeline {
	State.BindProgramPipeline(shaderPipeline.glId)
	return BoundShaderPipeline(shaderPipeline)
}

func (shaderPipeline *shaderPipeline) Id() uint32 {
	return shaderPipeline.glId
}

func (shaderPipeline *shaderPipeline) Delete() {
	gl.DeleteProgramPipelines(1, &shaderPipeline.glId)
	shaderPipeline.glId = 0
}

type glslDef struct {
	marker  string
	name    string
	value   string
	boolean bool
}

type program struct {
	uniformLocations map[string]int32
	definitions      map[string]glslDef
	versionEnd       int
	glId             uint32
	sourceTemplate   string
	sourceLive       string
	stage            int
}

type ShaderProgram interface {
	LabeledGlObject
	Id() uint32
	Compile() error
	CompileWith(defs map[string]string) error
	GetUniformLocation(name string) int32
	SetUniform(name string, value any)
	SetUniformIndexed(name string, index int, value any)
	Source() string
	Delete()
}

func NewShader(source string, stage int) ShaderProgram {
	// Currently unused
	// metaMatches := shaderMetaPattern.FindAllStringSubmatch(source, -1)
	// for _, match := range metaMatches {
	// 	key, value := match[1], strings.TrimSpace(match[2])
	// }

	defineMatches := shaderDefinePattern.FindAllStringSubmatch(source, -1)
	definitions := make(map[string]glslDef, len(defineMatches))
	defineMarkers := make(map[string]string, len(defineMatches))
	for i, match := range defineMatches {
		value := strings.TrimSpace(match[3])
		marker := fmt.Sprintf("$def_%v$", i)
		boolean := value == ""
		if boolean && match[1] == "//" {
			value = "false"
		}
		definitions[strings.ToLower(match[2])] = glslDef{
			marker:  marker,
			name:    match[2],
			value:   value,
			boolean: boolean,
		}
		defineMarkers[match[0]] = marker
	}
	source = shaderDefinePattern.ReplaceAllStringFunc(source, func(s string) string {
		return defineMarkers[s]
	})

	return &program{
		definitions:    definitions,
		stage:          stage,
		sourceTemplate: source,
		versionEnd:     shaderVersionPattern.FindStringIndex(source)[1],
	}
}

func (prog *program) Compile() error {
	return prog.CompileWith(nil)
}

func (prog *program) SetDebugLabel(label string) {
	// FIXME: Doesn't work
	setObjectLabel(gl.PROGRAM, prog.glId, label)
}

func (prog *program) CompileWith(defs map[string]string) error {
	source := prog.sourceTemplate

	for n, v := range defs {
		k := strings.ToLower(n)
		if def, ok := prog.definitions[k]; ok {
			sub := fmt.Sprintf("#define %v %v", def.name, v)
			if def.boolean {
				sub = fmt.Sprintf("#define %v", def.name)
			}
			if def.boolean && v == "false" {
				source = strings.Replace(source, def.marker, "// "+sub, 1)
			} else {
				source = strings.Replace(source, def.marker, sub, 1)
			}
		} else {
			source = source[:prog.versionEnd] + fmt.Sprintf("\n#define %v %v", n, v) + source[prog.versionEnd:]
		}
	}

	for _, def := range prog.definitions {
		sub := fmt.Sprintf("#define %v %v", def.name, def.value)
		if def.boolean {
			sub = fmt.Sprintf("#define %v", def.name)
		}
		if def.boolean && def.value == "false" {
			source = strings.Replace(source, def.marker, "// "+sub, 1)
		} else {
			source = strings.Replace(source, def.marker, sub, 1)
		}
	}

	var id uint32
	cStrs, free := gl.Strs(source + "\x00")
	id = gl.CreateShaderProgramv(uint32(prog.stage), 1, cStrs)
	free()

	var ok int32
	gl.GetProgramiv(id, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		return fmt.Errorf("failed to link shader, log: %v", readProgramInfoLog(id))
	}
	gl.ValidateProgram(id)
	gl.GetProgramiv(id, gl.VALIDATE_STATUS, &ok)
	if ok == gl.FALSE {
		return fmt.Errorf("failed to validate shader, log: %v", readProgramInfoLog(id))
	}

	prog.glId = id
	prog.sourceLive = source
	prog.uniformLocations = map[string]int32{}

	return nil
}

func (prog *program) Source() string {
	return prog.sourceLive
}

func (prog *program) Id() uint32 {
	return prog.glId
}

func (prog *program) Delete() {
	gl.DeleteProgram(prog.Id())
	prog.glId = 0
}

func readProgramInfoLog(id uint32) string {
	var logLength int32
	gl.GetProgramiv(id, gl.INFO_LOG_LENGTH, &logLength)

	log := strings.Repeat("\x00", int(logLength+1))
	gl.GetProgramInfoLog(id, logLength, nil, gl.Str(log))
	return log
}

func (prog *program) GetUniformLocation(name string) int32 {
	if location, ok := prog.uniformLocations[name]; ok {
		return location
	}

	location := gl.GetUniformLocation(prog.glId, gl.Str(name+"\x00"))
	prog.uniformLocations[name] = location

	if location == -1 {
		log.Printf("could not get location of %q\n", name)
	}

	return location
}

func (prog *program) SetUniformIndexed(name string, index int, value any) {
	location := prog.GetUniformLocation(name)
	if location == -1 {
		return
	}
	setProgramUniformAny(prog.glId, location+int32(index), value)
}

func (prog *program) SetUniform(name string, value any) {
	location := prog.GetUniformLocation(name)
	if location == -1 {
		return
	}
	setProgramUniformAny(prog.glId, location, value)
}

func setProgramUniformAny(prog uint32, location int32, value any) {
	for refVal := reflect.ValueOf(value); refVal.Kind() == reflect.Ptr; refVal = reflect.ValueOf(value) {
		value = refVal.Elem().Interface()
	}

	switch v := value.(type) {
	case float64:
		gl.ProgramUniform1d(prog, location, v)
	case float32:
		gl.ProgramUniform1f(prog, location, v)
	case int:
		gl.ProgramUniform1i(prog, location, int32(v))
	case int64:
		gl.ProgramUniform1i(prog, location, int32(v))
	case int32:
		gl.ProgramUniform1i(prog, location, int32(v))
	case int16:
		gl.ProgramUniform1i(prog, location, int32(v))
	case int8:
		gl.ProgramUniform1i(prog, location, int32(v))
	case uint:
		gl.ProgramUniform1ui(prog, location, uint32(v))
	case uint64:
		gl.ProgramUniform1ui(prog, location, uint32(v))
	case uint32:
		gl.ProgramUniform1ui(prog, location, uint32(v))
	case uint16:
		gl.ProgramUniform1ui(prog, location, uint32(v))
	case uint8:
		gl.ProgramUniform1ui(prog, location, uint32(v))
	case mgl32.Vec2:
		gl.ProgramUniform2f(prog, location, v.X(), v.Y())
	case mgl64.Vec2:
		gl.ProgramUniform2d(prog, location, v.X(), v.Y())
	case mgl32.Vec3:
		gl.ProgramUniform3f(prog, location, v.X(), v.Y(), v.Z())
	case mgl64.Vec3:
		gl.ProgramUniform3d(prog, location, v.X(), v.Y(), v.Z())
	case mgl32.Vec4:
		gl.ProgramUniform4f(prog, location, v.X(), v.Y(), v.Z(), v.W())
	case mgl64.Vec4:
		gl.ProgramUniform4d(prog, location, v.X(), v.Y(), v.Z(), v.W())
	case mgl32.Mat3:
		gl.ProgramUniformMatrix3fv(prog, location, 1, false, &v[0])
	case mgl64.Mat3:
		gl.ProgramUniformMatrix3dv(prog, location, 1, false, &v[0])
	case mgl32.Mat4:
		gl.ProgramUniformMatrix4fv(prog, location, 1, false, &v[0])
	case mgl64.Mat4:
		gl.ProgramUniformMatrix4dv(prog, location, 1, false, &v[0])
	default:
		reflectType := reflect.TypeOf(value)
		dataType := reflectType.String()
		log.Panicf("Unsupported type %v", dataType)
	}
}
