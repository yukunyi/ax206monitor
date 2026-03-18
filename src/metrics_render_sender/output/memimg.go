package output

type MemImgOutputHandler struct{}

func NewMemImgOutputHandler() *MemImgOutputHandler {
	return &MemImgOutputHandler{}
}

func (m *MemImgOutputHandler) GetType() string {
	return "memimg"
}

func (m *MemImgOutputHandler) OutputFrame(frame *OutputFrame) error {
	if frame == nil {
		return nil
	}
	data, err := frame.PNG()
	if err != nil {
		return err
	}
	SetMemImgPNG(data)
	return nil
}

func (m *MemImgOutputHandler) Close() error {
	return nil
}
