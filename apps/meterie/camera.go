package meterie

// camera.go: Camera mode — webcam / phone (IP camera) / image file sources,
// crop & calibration, and feeding OCR readings onto the timeline.

import (
	"bufio"
	"bytes"
	"errors"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// FrameSource delivers grey frames.
type FrameSource interface {
	Grab() (*image.Gray, error)
	Close()
}

var camUnits = []struct {
	label  string
	q      Quantity
	unit   string
	prefix string
	f      Flags
}{
	{"V DC", QVoltage, "V", "", FDC}, {"mV DC", QVoltage, "V", "m", FDC}, {"V AC", QVoltage, "V", "", FAC},
	{"A DC", QCurrent, "A", "", FDC}, {"mA DC", QCurrent, "A", "m", FDC}, {"µA DC", QCurrent, "A", "µ", FDC},
	{"Ω", QResistance, "Ω", "", 0}, {"kΩ", QResistance, "Ω", "k", 0}, {"MΩ", QResistance, "Ω", "M", 0},
	{"nF", QCapacitance, "F", "n", 0}, {"µF", QCapacitance, "F", "µ", 0}, {"Hz", QFrequency, "Hz", "", 0},
	{"kHz", QFrequency, "Hz", "k", 0}, {"°C", QTemperature, "°C", "", 0}, {"%", QDuty, "%", "", 0}, {"(none)", QUnitless, "", "", 0},
}

type CameraState struct {
	src        int // 0 webcam, 1 phone/IP, 2 image file
	camIdx     int
	cams       []string
	url        string
	file       string
	running    bool
	source     FrameSource
	mu         sync.Mutex
	frame      *image.Gray
	frameSeq   int
	seenSeq    int
	err        string
	p          OCRParams
	unitSel    int
	text       string
	ok         bool
	dbg        OCRDebug
	lastText   string
	agree      int
	feed       bool
	chanID     int
	calib      string
	drag       int // 0 none, 1..4 corners, 5 move
	dragX      int
	dragY      int
	dragCrop   [4]float64
	fps        float64
	threshAuto bool
	stop       chan struct{}
	lastPush   time.Time
}

func (c *CameraState) init() {
	c.p = OCRParams{Crop: [4]float64{0.2, 0.3, 0.8, 0.6}, Digits: 4, Thick: 0.22, Invert: true, DP: true}
	c.url = "http://192.168.1.50:8080/shot.jpg"
	c.threshAuto = true
	c.fps = 4
}

func (c *CameraState) Start() {
	if c.running {
		return
	}
	var src FrameSource
	var err error
	switch c.src {
	case 0:
		src, err = platformOpenCamera(c.camIdx)
	case 1:
		src, err = newURLSource(c.url)
	case 2:
		src, err = newFileSource(c.file)
	}
	if err != nil {
		c.err = err.Error()
		return
	}
	c.err = ""
	c.source = src
	c.running = true
	c.stop = make(chan struct{})
	go c.loop(src, c.stop)
}

func (c *CameraState) Stop() {
	if !c.running {
		return
	}
	close(c.stop)
	c.running = false
}

func (c *CameraState) loop(src FrameSource, stop chan struct{}) {
	defer src.Close()
	interval := time.Duration(1e9 / math.Max(0.5, c.fps))
	for {
		select {
		case <-stop:
			return
		default:
		}
		t0 := time.Now()
		g, err := src.Grab()
		c.mu.Lock()
		if err != nil {
			c.err = err.Error()
		} else if g != nil {
			c.frame = g
			c.frameSeq++
			c.err = ""
		}
		c.mu.Unlock()
		appWake()
		if d := interval - time.Since(t0); d > 0 {
			select {
			case <-stop:
				return
			case <-time.After(d):
			}
		}
	}
}

// ---------------------------------------------------------------------------
// URL source: snapshot JPEG polling or MJPEG stream (IP Webcam / DroidCam style apps)

type urlSource struct {
	url    string
	client *http.Client
	mr     *multipart.Reader
	body   io.ReadCloser
}

func newURLSource(u string) (FrameSource, error) {
	u = strings.TrimSpace(u)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return nil, errors.New("enter the camera URL, e.g. http://192.168.1.50:8080/shot.jpg")
	}
	return &urlSource{url: u, client: &http.Client{Timeout: 8 * time.Second}}, nil
}

func (s *urlSource) Grab() (*image.Gray, error) {
	if s.mr != nil {
		p, err := s.mr.NextPart()
		if err != nil {
			s.body.Close()
			s.mr = nil
			return nil, err
		}
		img, _, err := image.Decode(bufio.NewReader(p))
		if err != nil {
			return nil, err
		}
		return toGray(img), nil
	}
	resp, err := s.client.Get(s.url)
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	mt, params, _ := mime.ParseMediaType(ct)
	if strings.HasPrefix(mt, "multipart/") && params["boundary"] != "" {
		s.client.Timeout = 0
		s.body = resp.Body
		s.mr = multipart.NewReader(resp.Body, params["boundary"])
		return s.Grab()
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, errors.New("the URL didn't return a JPEG/PNG image (try …/shot.jpg or …/video)")
	}
	return toGray(img), nil
}

func (s *urlSource) Close() {
	if s.body != nil {
		s.body.Close()
	}
}

type fileSource struct{ img *image.Gray }

func newFileSource(p string) (FrameSource, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, errors.New("choose a photo of the meter display first")
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return &fileSource{toGray(img)}, nil
}
func (s *fileSource) Grab() (*image.Gray, error) { return s.img, nil }
func (s *fileSource) Close()                     {}

func toGray(img image.Image) *image.Gray {
	if g, ok := img.(*image.Gray); ok {
		return g
	}
	b := img.Bounds()
	// downscale very large photos for speed
	scale := 1
	for b.Dx()/scale > 1600 {
		scale++
	}
	g := image.NewGray(image.Rect(0, 0, b.Dx()/scale, b.Dy()/scale))
	if scale == 1 {
		draw.Draw(g, g.Bounds(), img, b.Min, draw.Src)
		return g
	}
	for y := 0; y < g.Rect.Dy(); y++ {
		for x := 0; x < g.Rect.Dx(); x++ {
			r, gg, bb, _ := img.At(b.Min.X+x*scale, b.Min.Y+y*scale).RGBA()
			g.Pix[y*g.Stride+x] = uint8((299*r + 587*gg + 114*bb) / 1000 >> 8)
		}
	}
	return g
}

// ---------------------------------------------------------------------------
// Page

func (a *App) pageCamera(r Rect) {
	u := a.ui
	c := &a.cam
	gap := u.Px(16)
	ctrlW := clampi(r.W*36/100, u.Px(340), u.Px(440))
	left, right := r.CutLeft(r.W - ctrlW - gap)
	right.X += gap
	right.W -= gap

	// grab latest frame and run OCR
	c.mu.Lock()
	frame := c.frame
	seq := c.frameSeq
	errS := c.err
	c.mu.Unlock()
	if c.threshAuto {
		c.p.Thresh = 0
	}
	if frame != nil && seq != c.seenSeq {
		c.seenSeq = seq
		c.text, c.ok, c.dbg = OCR(frame, c.p)
		if c.ok && c.text == c.lastText {
			c.agree++
		} else {
			c.agree = 0
		}
		c.lastText = c.text
		if c.feed && c.ok && c.agree >= 1 {
			a.pushCameraReading()
		}
	}

	// preview
	in := u.Card(left, "Camera view — drag the box around the digits")
	view := Rect{in.X, in.Y, in.W, in.H - u.Px(10)}
	u.C.FillRoundRect(view, u.Pf(10), th.Bg2)
	if frame == nil {
		msg := "Choose a source on the right and press Start."
		if errS != "" {
			msg = errS
		}
		u.Text(icCamera, Rect{view.X, view.CY() - u.Px(50), view.W, u.Px(40)}, u.FI(30), th.Faint, AlignCenter)
		u.Wrap(msg, view.X+u.Px(40), view.CY(), view.W-u.Px(80), u.F(13.5), th.Faint)
	} else {
		fw, fh := frame.Rect.Dx(), frame.Rect.Dy()
		sc := math.Min(float64(view.W)/float64(fw), float64(view.H)/float64(fh))
		iw, ih := int(float64(fw)*sc), int(float64(fh)*sc)
		ir := Rect{view.X + (view.W-iw)/2, view.Y + (view.H-ih)/2, iw, ih}
		u.PushClip(view)
		u.C.DrawImage(frame, ir)
		a.drawCropOverlay(ir)
		u.PopClip()
	}

	// controls
	cin := u.Card(right, "")
	off := u.BeginScroll("cam#ctl", cin)
	y := cin.Y - off
	u.T.Draw("Source", cin.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	if u.Segmented("cam#src", Rect{cin.X, y, cin.W, u.Px(32)}, []string{"Webcam", "Phone / IP", "Photo"}, &c.src) {
		c.Stop()
	}
	y += u.Px(44)
	switch c.src {
	case 0:
		if c.cams == nil {
			c.cams = platformListCameras()
			if len(c.cams) == 0 {
				c.cams = []string{"(no webcam found)"}
			}
		}
		u.Dropdown("cam#dev", Rect{cin.X, y, cin.W - u.Px(44), u.Px(36)}, c.cams, &c.camIdx)
		if u.IconButton("cam#rescan", Rect{cin.Right() - u.Px(36), y, u.Px(36), u.Px(36)}, icRefresh, "Find cameras", false) {
			c.cams = nil
		}
		y += u.Px(46)
	case 1:
		u.TextField("cam#url", Rect{cin.X, y, cin.W, u.Px(36)}, &c.url, "http://phone-ip:8080/shot.jpg")
		y += u.Px(42)
		y += u.Wrap("Use a phone camera app that serves JPEG snapshots or MJPEG (e.g. 'IP Webcam' on Android: …/shot.jpg or …/video). PC and phone must be on the same Wi-Fi.", cin.X, y, cin.W, u.F(11.5), th.Faint) + u.Px(8)
	case 2:
		fn := c.file
		if fn == "" {
			fn = "No photo chosen"
		}
		u.Text(fn, Rect{cin.X, y, cin.W - u.Px(110), u.Px(36)}, u.F(12.5), th.Text, AlignLeft)
		if u.Button("cam#pick", Rect{cin.Right() - u.Px(100), y, u.Px(100), u.Px(36)}, icOpen, "Choose", BtnSecondary) {
			if p, ok := platformOpenDialog("Choose a photo of the display", "Images (*.jpg;*.png)", "*.jpg;*.jpeg;*.png"); ok {
				c.file = p
				c.Stop()
				c.Start()
			}
		}
		y += u.Px(46)
	}
	if !c.running {
		if u.Button("cam#start", Rect{cin.X, y, cin.W, u.Px(40)}, icPlay, "Start camera", BtnPrimary) {
			c.Start()
			if c.err != "" {
				u.Toast(c.err, 3)
			}
		}
	} else if u.Button("cam#stop", Rect{cin.X, y, cin.W, u.Px(40)}, icStop, "Stop camera", BtnSecondary) {
		c.Stop()
	}
	y += u.Px(56)
	// reading
	res := Rect{cin.X, y, cin.W, u.Px(84)}
	u.C.FillRoundRect(res, u.Pf(12), th.LCD)
	col := th.Faint
	txt := "—"
	if c.text != "" {
		txt = c.text
		col = th.Warn
		if c.ok {
			col = th.LCDOn
		}
	}
	u.T.Draw("Reads as", res.X+u.Px(14), res.Y+u.Px(10), u.F(11.5), th.Sub)
	u.T.Draw(txt+"  "+camUnits[c.unitSel].label, res.X+u.Px(14), res.Y+u.Px(28), u.FH(30), col)
	y += u.Px(96)
	u.T.Draw("Digits on the display", cin.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	ds := c.p.Digits - 3
	if u.Segmented("cam#digits", Rect{cin.X, y, cin.W, u.Px(32)}, []string{"3", "4", "5", "6"}, &ds) {
		c.p.Digits = ds + 3
	}
	y += u.Px(44)
	dt := b2i(!c.p.Invert)
	if u.Segmented("cam#type", Rect{cin.X, y, cin.W, u.Px(32)}, []string{"LCD (dark digits)", "LED / backlit"}, &dt) {
		c.p.Invert = dt == 0
	}
	y += u.Px(44)
	slider := func(id, label string, v *float64, lo, hi float64, show string) {
		u.Text(label, Rect{cin.X, y, cin.W / 2, u.Px(22)}, u.F(12.5), th.Text, AlignLeft)
		u.Text(show, Rect{cin.X + cin.W/2, y, cin.W / 2, u.Px(22)}, u.F(12.5), th.Sub, AlignRight)
		y += u.Px(24)
		u.Slider(id, Rect{cin.X + u.Px(8), y, cin.W - u.Px(16), u.Px(22)}, v, lo, hi)
		y += u.Px(34)
	}
	slider("cam#skew", "Slant (italic digits)", &c.p.Skew, -0.35, 0.35, itoa(int(c.p.Skew*100))+"%")
	slider("cam#thick", "Segment thickness", &c.p.Thick, 0.08, 0.38, itoa(int(c.p.Thick*100))+"%")
	u.Toggle("cam#autothr", Rect{cin.X, y, cin.W, u.Px(28)}, &c.threshAuto, "Automatic brightness threshold")
	y += u.Px(36)
	if !c.threshAuto {
		if c.p.Thresh == 0 {
			c.p.Thresh = 0.5
		}
		slider("cam#thr", "Threshold", &c.p.Thresh, 0.05, 0.95, itoa(int(c.p.Thresh*100))+"%")
	}
	u.Toggle("cam#dp", Rect{cin.X, y, cin.W, u.Px(28)}, &c.p.DP, "Detect decimal points")
	y += u.Px(40)
	u.T.Draw("Calibrate: type what the display shows now", cin.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	u.TextField("cam#calib", Rect{cin.X, y, cin.W - u.Px(112), u.Px(36)}, &c.calib, "e.g. 12.34")
	if u.ButtonEx("cam#calbtn", Rect{cin.Right() - u.Px(104), y, u.Px(104), u.Px(36)}, icSync, "Auto-fit", BtnSecondary, frame != nil && c.calib != "") {
		q, found := OCRCalibrate(frame, c.p, c.calib)
		c.p = q
		c.threshAuto = q.Thresh == 0
		if found {
			u.Toast("Calibrated — the reader now matches "+c.calib, 1)
		} else {
			u.Toast("Couldn't match exactly; adjust the box so it hugs the digits", 2)
		}
	}
	y += u.Px(50)
	u.T.Draw("Unit shown on the display", cin.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	opts := make([]string, len(camUnits))
	for i, cu := range camUnits {
		opts[i] = cu.label
	}
	u.Dropdown("cam#unit", Rect{cin.X, y, cin.W, u.Px(36)}, opts, &c.unitSel)
	y += u.Px(50)
	if u.Toggle("cam#feed", Rect{cin.X, y, cin.W, u.Px(30)}, &c.feed, "Send readings to the timeline") && c.feed {
		if a.chanByID(c.chanID) == nil {
			d := &Device{ID: a.nextDevID, Kind: TrCamera, Label: "Camera"}
			a.nextDevID++
			ch := a.addChannel("Camera", KindCamera, "Camera OCR", d.ID)
			d.ChannelID = ch.ID
			a.devices = append(a.devices, d)
			d.Start()
			c.chanID = ch.ID
		}
	}
	y += u.Px(40)
	y += u.Wrap("Tips: fill the frame with the display, avoid glare (tilt the meter slightly), and keep the camera still. Readings are only accepted when two frames in a row agree.", cin.X, y, cin.W, u.F(11.5), th.Faint)
	y += u.Px(10)
	u.EndScroll("cam#ctl", cin, y+off-cin.Y)
}

func (a *App) pushCameraReading() {
	c := &a.cam
	ch := a.chanByID(c.chanID)
	if ch == nil || time.Since(c.lastPush) < 150*time.Millisecond {
		return
	}
	v, ok, ol := OCRValue(c.text)
	if !ok {
		return
	}
	cu := camUnits[c.unitSel]
	r := Reading{T: time.Now(), Q: cu.q, Unit: cu.unit, Prefix: cu.prefix, Flags: cu.f, Display: c.text, Note: "camera"}
	r.Counts = int(math.Pow(10, float64(c.p.Digits))) - 1
	if ol {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
	} else {
		r.Value = v * prefixMult(cu.prefix)
	}
	c.lastPush = time.Now()
	a.ingest(ch, r)
}

func (a *App) drawCropOverlay(ir Rect) {
	u := a.ui
	c := &a.cam
	cp := &c.p.Crop
	px := func(fx, fy float64) (int, int) { return ir.X + int(fx*float64(ir.W)), ir.Y + int(fy*float64(ir.H)) }
	x0, y0 := px(cp[0], cp[1])
	x1, y1 := px(cp[2], cp[3])
	box := Rect{x0, y0, x1 - x0, y1 - y0}
	// dim outside
	dim := RGBA(5, 8, 12, 130)
	u.C.FillRect(Rect{ir.X, ir.Y, ir.W, box.Y - ir.Y}, dim)
	u.C.FillRect(Rect{ir.X, box.Bottom(), ir.W, ir.Bottom() - box.Bottom()}, dim)
	u.C.FillRect(Rect{ir.X, box.Y, box.X - ir.X, box.H}, dim)
	u.C.FillRect(Rect{box.Right(), box.Y, ir.Right() - box.Right(), box.H}, dim)
	u.C.StrokeRoundRect(box, u.Pf(4), u.Pf(2), th.Accent)
	// cells + sample points
	n := c.p.Digits
	cw := float64(box.W) / float64(n)
	for i := 0; i < n; i++ {
		cx := float64(box.X) + float64(i)*cw
		if i > 0 {
			u.C.DashedVLine(int(cx), box.Y, box.Bottom(), u.Px(3), u.Px(3), th.Accent.WithA(0.4))
		}
		for s := 0; s < 8; s++ {
			if s == 7 && !c.p.DP {
				continue
			}
			ss := segSample[s]
			sy := float64(box.Y) + ss[1]*float64(box.H)
			skew := c.p.Skew * cw * (1 - ss[1])
			sx := cx + ss[0]*cw + skew
			lit := false
			if i < len(c.dbg.Cells) {
				lit = c.dbg.Cells[i][s] > 0.45
			}
			col := th.Faint
			if lit {
				col = th.Good
			}
			u.C.FillCircle(sx, sy, u.Pf(3.2), col)
			u.C.StrokeCircle(sx, sy, u.Pf(4.2), u.Pf(1), th.Bg)
		}
	}
	// handles
	hs := u.Px(10)
	corners := [][2]int{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
	for _, k := range corners {
		u.C.FillRoundRect(Rect{k[0] - hs/2, k[1] - hs/2, hs, hs}, u.Pf(3), th.Text)
	}
	// interaction
	in := u.In
	if u.Hover(ir) {
		u.hot = "cam#crop"
		u.Cursor = CurCross
		if in.Pressed {
			c.drag = 5
			for i, k := range corners {
				if abs(in.MX-k[0]) < hs*2 && abs(in.MY-k[1]) < hs*2 {
					c.drag = i + 1
				}
			}
			if c.drag == 5 && !box.Contains(in.MX, in.MY) {
				// start a fresh box
				fx := float64(in.MX-ir.X) / float64(ir.W)
				fy := float64(in.MY-ir.Y) / float64(ir.H)
				*cp = [4]float64{fx, fy, fx + 0.01, fy + 0.01}
				c.drag = 3
			}
			c.dragX, c.dragY, c.dragCrop = in.MX, in.MY, *cp
			u.active = "cam#crop"
		}
	}
	if c.drag != 0 && in.Down && u.active == "cam#crop" {
		dx := float64(in.MX-c.dragX) / float64(ir.W)
		dy := float64(in.MY-c.dragY) / float64(ir.H)
		o := c.dragCrop
		switch c.drag {
		case 1:
			cp[0], cp[1] = o[0]+dx, o[1]+dy
		case 2:
			cp[2], cp[1] = o[2]+dx, o[1]+dy
		case 3:
			cp[2], cp[3] = o[2]+dx, o[3]+dy
		case 4:
			cp[0], cp[3] = o[0]+dx, o[3]+dy
		case 5:
			w, h := o[2]-o[0], o[3]-o[1]
			cp[0] = clampf(o[0]+dx, 0, 1-w)
			cp[1] = clampf(o[1]+dy, 0, 1-h)
			cp[2], cp[3] = cp[0]+w, cp[1]+h
		}
		for i := range cp {
			cp[i] = clamp01(cp[i])
		}
		if cp[2] < cp[0]+0.02 {
			cp[2] = cp[0] + 0.02
		}
		if cp[3] < cp[1]+0.02 {
			cp[3] = cp[1] + 0.02
		}
		c.seenSeq = -1 // re-run OCR
		u.Cursor = CurMove
	}
	if in.Released {
		c.drag = 0
	}
}
