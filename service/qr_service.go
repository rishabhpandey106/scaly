package service

import (
	"bytes"
	"image/color"

	qrcode "github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type QRService struct{}

func NewQRService() *QRService {
	return &QRService{}
}

type writeCloser struct {
	*bytes.Buffer
}

func (wc writeCloser) Close() error {
	return nil
}

func (s *QRService) Generate(url string) ([]byte, error) {
	qrc, err := qrcode.New(url)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	wc := writeCloser{buf}

	// grad := &standard.LinearGradient{
	// 	Angle: 0,
	// 	Stops: []standard.ColorStop{
	// 		{T: 0.0, Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}}, // red
	// 		{T: 0.5, Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}}, // green
	// 		{T: 1.0, Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}}, // blue
	// 	},
	// }

	w := standard.NewWithWriter(wc,
		standard.WithBgColor(color.White),
		standard.WithFgColor(color.RGBA{R: 255, G: 156, B: 22, A: 0.8 * 255}),
		standard.WithCircleShape(),
		// standard.WithFgGradient(grad),
		standard.WithLogoImageFilePNG("./assets/logo.png"),
		standard.WithBuiltinImageEncoder(standard.PNG_FORMAT),
		standard.WithLogoSafeZone(),
	)

	if err := qrc.Save(w); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
