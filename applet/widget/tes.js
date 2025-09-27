function clamp(v, a, b) { return Math.max(a, Math.min(b, v)); }
function px(n) { return Math.round(n); }

let robotoData;

// Basic EventEmitter
class Emitter {
  constructor() { this._listeners = new Map(); }
  on(ev, fn) { (this._listeners.get(ev) || this._listeners.set(ev, []).get(ev)).push(fn); return () => this.off(ev, fn); }
  off(ev, fn) { const arr = this._listeners.get(ev); if (!arr) return; const i = arr.indexOf(fn); if (i >= 0) arr.splice(i, 1); }
  emit(ev, ...args) { (this._listeners.get(ev) || []).slice().forEach(fn => fn(...args)); }
}

// ---------- Core GUI classes ----------
export default class SkiaGUI extends Emitter {
  constructor(canvasEl, Skia) {
    super();
    this.canvas = canvasEl;
    this.Skia = Skia;
    this.root = null;
    this._dpi = window.devicePixelRatio || 1;
    this._running = false;
    this._lastTime = 0;
    this._raf = null;
    this._pointer = { x: 0, y: 0, down: false };
    this.surface = null;
    this.canvasWidth = 0; this.canvasHeight = 0;
    this._onPointerMove = this._onPointerMove.bind(this);
    this._onPointerDown = this._onPointerDown.bind(this);
    this._onPointerUp = this._onPointerUp.bind(this);
    this._onResize = this._onResize.bind(this);
  }

  async init() {
    const Skia = this.Skia;
    this._updateCanvasSize();
    try {
      if (Skia && Skia.MakeWebGLCanvasSurface) {
        this.surface = Skia.MakeWebGLCanvasSurface(this.canvas);
      } else if (Skia && Skia.Surface && Skia.Surface.MakeFromCanvas) {
        this.surface = Skia.Surface.MakeFromCanvas(this.canvas);
      } else if (Skia && Skia.MakeCanvasSurface) {
        this.surface = Skia.MakeCanvasSurface(this.canvas);
      } else {
        const width = this.canvasWidth, height = this.canvasHeight;
        this.surface = Skia.Surface.MakeRasterN32Premul(width, height);
        this._rasterFallback = true;
      }
    } catch (err) {
      console.warn('Skia surface creation failed, falling back to raster surface or native 2D; error:', err);
      const width = this.canvasWidth, height = this.canvasHeight;
      this.surface = Skia.Surface.MakeRasterN32Premul(width, height);
      this._rasterFallback = true;
    }
    if (!this.surface) throw new Error('Failed to create Skia surface');

    this.canvas.style.touchAction = 'none';
    this.canvas.addEventListener('pointermove', this._onPointerMove);
    this.canvas.addEventListener('pointerdown', this._onPointerDown);
    window.addEventListener('pointerup', this._onPointerUp);
    window.addEventListener('resize', this._onResize);

    this._running = true;
    this._lastTime = performance.now();
    this._raf = requestAnimationFrame(this._frame.bind(this));
  }

  _updateCanvasSize() {
    const cssW = this.canvas.clientWidth || this.canvas.width || 300;
    const cssH = this.canvas.clientHeight || this.canvas.height || 150;
    this._dpi = window.devicePixelRatio || 1;
    const w = Math.max(1, Math.round(cssW * this._dpi));
    const h = Math.max(1, Math.round(cssH * this._dpi));
    if (this.canvas.width !== w || this.canvas.height !== h) {
      this.canvas.width = w;
      this.canvas.height = h;
    }
    this.canvasWidth = w; this.canvasHeight = h;
  }

  setRoot(widget) { this.root = widget; if (this.root) this.root._setGUI(this); return this; }

  destroy() {
    this._running = false;
    if (this._raf) cancelAnimationFrame(this._raf);
    this.canvas.removeEventListener('pointermove', this._onPointerMove);
    this.canvas.removeEventListener('pointerdown', this._onPointerDown);
    window.removeEventListener('pointerup', this._onPointerUp);
    window.removeEventListener('resize', this._onResize);
  }

  _onPointerMove(e) {
    const rect = this.canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left) * this._dpi;
    const y = (e.clientY - rect.top) * this._dpi;
    this._pointer.x = x; this._pointer.y = y;
    if (this.root) this.root._propagateEvent({ type: 'pointermove', x, y, originalEvent: e });
  }
  _onPointerDown(e) {
    const rect = this.canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left) * this._dpi;
    const y = (e.clientY - rect.top) * this._dpi;
    this._pointer.down = true; this._pointer.x = x; this._pointer.y = y;
    if (this.root) this.root._propagateEvent({ type: 'pointerdown', x, y, originalEvent: e });
  }
  _onPointerUp(e) {
    const rect = this.canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left) * this._dpi;
    const y = (e.clientY - rect.top) * this._dpi;
    this._pointer.down = false; this._pointer.x = x; this._pointer.y = y;
    if (this.root) this.root._propagateEvent({ type: 'pointerup', x, y, originalEvent: e });
  }
  _onResize() { this._updateCanvasSize(); }

  _frame(now) {
    const dt = now - this._lastTime; this._lastTime = now;
    if (!this._running) return;
    this._render(dt);
    this._raf = requestAnimationFrame(this._frame.bind(this));
  }

  _render(dt) {
    const canvas = this.surface.getCanvas();
    canvas.clear(this.Skia.WHITE);
    if (this.root) {
      this.root._layout(0, 0, this.canvasWidth, this.canvasHeight, this.Skia);
      this.root._draw(canvas, this.Skia, dt);
    }
    this.surface.flush();
  }
}

// ---------- Widget system ----------
export class Widget extends Emitter {
  constructor(props = {}) {
    super();
    this.x = props.x ?? 0; this.y = props.y ?? 0; this.w = props.w ?? 0; this.h = props.h ?? 0;
    this.parent = null; this.children = [];
    this.props = props;
    this._gui = null;
    this.visible = true;
  }
  _setGUI(gui) { this._gui = gui; this.children.forEach(c => c._setGUI(gui)); }
  append(child) { child.parent = this; child._setGUI(this._gui); this.children.push(child); return this; }
  remove(child) { const i = this.children.indexOf(child); if (i >= 0) this.children.splice(i, 1); child.parent = null; }

  // layout: set x,y,w,h (in device pixels)
  _layout(x, y, w, h, Skia) {
    this.x = x; this.y = y; this.w = w; this.h = h;
    // default: stack children to fill
    let cx = x, cy = y;
    this.children.forEach(ch => ch._layout(cx, cy, w, h, Skia));
  }

  _draw(canvas, Skia, dt) {
    if (!this.visible) return;
    // default draws children
    this.children.forEach(ch => ch._draw(canvas, Skia, dt));
  }

  _propagateEvent(ev) {
    if (!this.visible) return false;
    // children in reverse paint order (topmost first)
    for (let i = this.children.length - 1; i >= 0; i--) {
      const ch = this.children[i];
      if (ch._containsPoint(ev.x, ev.y)) {
        if (ch._propagateEvent(ev)) return true;
      }
    }
    // Not handled by children -> this widget
    return this._handleEvent && this._handleEvent(ev);
  }

  _containsPoint(px, py) { return px >= this.x && py >= this.y && px <= this.x + this.w && py <= this.y + this.h; }
}

export class VBox extends Widget {
  constructor(props = {}, children = []) { super(props); this.x = props.x ?? 0; this.y = props.y ?? 0; this.w = props.w ?? 0; this.h = props.h ?? 0; this.children = children; this.spacing = props.spacing || 6; children.forEach(c => c.parent = this); }
  _layout(x, y, w, h, Skia) {
    let cy = y;
    this.children.forEach(ch => {
      ch._layout(x, cy, ch.w, ch.h, Skia);
      cy += ch.h + this.spacing;
    });
  }
}

export class HBox extends Widget {
  constructor(props = {}, children = []) { super(props); this.children = children; this.spacing = props.spacing || 6; children.forEach(c => c.parent = this); }
  _layout(x, y, w, h, Skia) {
    this.x = x; this.y = y; this.w = w; this.h = h;
    const totalSpacing = this.spacing * Math.max(0, this.children.length - 1);
    const childW = Math.max(0, (w - totalSpacing) / Math.max(1, this.children.length));
    let cx = x;
    for (const ch of this.children) {
      ch._layout(cx, y, childW, h, Skia);
      cx += childW + this.spacing;
    }
  }
}

export class Label extends Widget {
  constructor(text, props = {}) { super(props); this.text = text; this.fontSize = props.fontSize || 14; this.fontFamily = props.fontFamily || 'sans-serif'; this.align = props.align || 'left'; }
  _draw(canvas, Skia) {
    const fontMgr = Skia.FontMgr.FromData([robotoData]);
    const paraStyle = new Skia.ParagraphStyle({
      textStyle: {
        color: Skia.BLACK,
        fontFamilies: [this.fontFamily],
        fontSize: this.fontSize,
      },
      textAlign: Skia.TextAlign.Left,
    });
    const builder = Skia.ParagraphBuilder.Make(paraStyle, fontMgr);
    builder.addText(this.text);
    const paragraph = builder.build();
    paragraph.layout(this.w);
    canvas.drawParagraph(paragraph, this.x, this.y);
  }
}

function RippleState(startX, startY, maxRadius, buttonBounds, startTime) {
  this.x = startX;
  this.y = startY;
  this.maxR = maxRadius;
  this.bounds = buttonBounds;
  this.startTime = startTime;
  this.duration = 400;
}

export class Button extends Widget {
  constructor(text, oc = null, props = {}) {
    super(props);
    this.text = text;
    this.bgColor = [0.4, 0.2, 0.9, 1.0];
    this.textColor = [1.0, 1.0, 1.0, 1.0];
    this.radius = props.radius ?? 16;
    this.elevation = 8;
    this.onClick = function (event) {
      const maxR = this.calculateMaxRadius(this.buttonRect, event.x, event.y);
      this.rippleEffect = new RippleState(event.x, event.y, maxR, this.buttonRect, performance.now());
      oc(event);
    }
    this._hover = false;
    this._pressed = false;
    this.rippleEffect = null;
  }

  calculateMaxRadius(rect, centerX, centerY) {
    const dx1 = rect.fLeft - centerX;
    const dy1 = rect.fTop - centerY;
    const dx2 = rect.fRight - centerX;
    const dy2 = rect.fBottom - centerY;

    return Math.sqrt(
      Math.max(
        dx1 * dx1 + dy1 * dy1,
        dx2 * dx2 + dy1 * dy1,
        dx1 * dx1 + dy2 * dy2,
        dx2 * dx2 + dy2 * dy2
      )
    );
  }

  _draw(canvas, Skia, currentTime) {

    let shadowOffsetX = 0;
    let shadowOffsetY = this.elevation * 0.7;
    let blurSigma = this.elevation * 0.7;

    if (blurSigma < 0.1) blurSigma = 0.1;

    const shadowColor = Skia.Color4f(0, 0, 0, 0.2 + (this.elevation * 0.01));

    this.buttonRect = Skia.LTRBRect(this.x, this.y, this.x + this.w, this.y + this.h);
    const rrect = Skia.RRectXY(this.buttonRect, this.radius, this.radius);

    if (this.elevation > 0) {
      const shadowPaint = new Skia.Paint();
      shadowPaint.setColor(shadowColor);
      shadowPaint.setStyle(Skia.PaintStyle.Fill);
      shadowPaint.setAntiAlias(true);

      const blurMaskFilter = Skia.MaskFilter.MakeBlur(
        Skia.BlurStyle.Normal,
        blurSigma,
        false
      );
      shadowPaint.setMaskFilter(blurMaskFilter);

      canvas.save();
      canvas.translate(shadowOffsetX, shadowOffsetY);
      canvas.drawRRect(rrect, shadowPaint);
      canvas.restore();

      blurMaskFilter.delete();
      shadowPaint.delete();
    }

    const fillPaint = new Skia.Paint();
    fillPaint.setColor(Skia.Color4f(this.bgColor[0], this.bgColor[1], this.bgColor[2], this.bgColor[3]));
    fillPaint.setStyle(Skia.PaintStyle.Fill);
    fillPaint.setAntiAlias(true);
    canvas.drawRRect(rrect, fillPaint);
    fillPaint.delete();

    const fontMgr = Skia.FontMgr.FromData([robotoData]);
    const paraStyle = new Skia.ParagraphStyle({
      textStyle: {
        color: Skia.Color4f(this.textColor[0], this.textColor[1], this.textColor[2], this.textColor[3]),
        fontFamilies: ["roboto"],
        fontSize: 16,
      },
      textAlign: Skia.TextAlign.Center,
    });
    const builder = Skia.ParagraphBuilder.Make(paraStyle, fontMgr);
    builder.addText(this.text);
    const paragraph = builder.build();
    paragraph.layout(this.w);
    canvas.drawParagraph(paragraph, this.x, (this.y + this.y + this.h - 24) / 2);

    const RIPPLE_COLOR = Skia.Color4f(1.0, 1.0, 1.0, 0.3);

    function drawRipple(canvas, CanvasKit, ripple, currentTime, radius) {
      const elapsed = currentTime - ripple.startTime;
      const progress = Math.min(1.0, elapsed / ripple.duration);

      const currentRadius = ripple.maxR * progress;
      const currentAlpha = (1.0 - progress) * 0.3;

      if (currentAlpha <= 0) return true;

      const ripplePaint = new CanvasKit.Paint();
      const color = [RIPPLE_COLOR[0], RIPPLE_COLOR[1], RIPPLE_COLOR[2], currentAlpha];
      ripplePaint.setColor(CanvasKit.Color4f(...color));
      ripplePaint.setStyle(CanvasKit.PaintStyle.Fill);
      ripplePaint.setAntiAlias(true);

      const clipRRect = CanvasKit.RRectXY(ripple.bounds, radius, radius);
      canvas.save();
      canvas.clipRRect(clipRRect, CanvasKit.ClipOp.Intersect, true);

      canvas.drawCircle(ripple.x, ripple.y, currentRadius, ripplePaint);

      canvas.restore();

      return false;
    }

    if (this.rippleEffect) {
      const finished = drawRipple(canvas, Skia, this.rippleEffect, currentTime, this.radius);
      if (finished) {
        this.rippleEffect = null;
      }
    }
  }

  _handleEvent(ev) {
    if (ev.type === 'pointermove') {
      const inside = this._containsPoint(ev.x, ev.y);
      if (inside !== this._hover) { this._hover = inside; this.emit('hover', inside); }
      return false;
    }
    if (ev.type === 'pointerdown') {
      if (this._containsPoint(ev.x, ev.y)) { this._pressed = true; return true; }
      return false;
    }
    if (ev.type === 'pointerup') {
      if (this._pressed && this._containsPoint(ev.x, ev.y)) {
        this._pressed = false;
        if (this.onClick) this.onClick(ev);
        this.emit('click');
        return true;
      }
      this._pressed = false;
      return false;
    }
    return false;
  }
}

export class Slider extends Widget {
  constructor(value = 0.5, onChange = null, props = {}) {
    super(props);
    this.value = value;
    this.onChange = onChange;
    this._dragging = false;
  }
  _draw(canvas, Skia) {
    const paint = new Skia.Paint(); paint.setAntiAlias(true);
    paint.setColor(Skia.Color4f(0.8, 0.8, 0.8, 1));
    canvas.drawRect({ x: this.x + 4, y: this.y + this.h / 2 - 2, width: this.w - 8, height: 4 }, paint);
    const knobX = this.x + 4 + this.value * (this.w - 8);
    paint.setColor(Skia.Color4f(0.3, 0.3, 0.8, 1));
    canvas.drawCircle(knobX, this.y + this.h / 2, Math.min(this.h / 2 - 2, 8), paint);
  }
  _handleEvent(ev) {
    if (ev.type === 'pointerdown' && this._containsPoint(ev.x, ev.y)) { this._dragging = true; return true; }
    if (ev.type === 'pointerup') { this._dragging = false; return false; }
    if (ev.type === 'pointermove' && this._dragging) {
      this.value = clamp((ev.x - this.x - 4) / (this.w - 8), 0, 1);
      if (this.onChange) this.onChange(this.value);
      this.emit('change', this.value);
      return true;
    }
    return false;
  }
}

export class Checkbox extends Widget {
  constructor(label = '', checked = false, onChange = null, props = {}) {
    super(props);
    this.label = label;
    this.checked = checked;
    this.onChange = onChange;
    this._hover = false;
  }
  _draw(canvas, Skia) {
    const size = Math.min(this.h, this.w, 20);
    const box = { x: this.x, y: this.y + (this.h - size) / 2, width: size, height: size };
    const paint = new Skia.Paint(); paint.setAntiAlias(true);
    paint.setColor(Skia.Color4f(0.95, 0.95, 0.95, 1));
    canvas.drawRect(box, paint);
    const border = new Skia.Paint(); border.setStyle(Skia.PaintStyle.Stroke); border.setStrokeWidth(1);
    border.setColor(Skia.Color4f(0.3, 0.3, 0.3, 1));
    canvas.drawRect(box, border);
    if (this.checked) {
      const mark = new Skia.Paint(); mark.setColor(Skia.Color4f(0.1, 0.4, 0.1, 1));
      canvas.drawLine(box.x + 3, box.y + size / 2, box.x + size / 2, box.y + size - 3, mark);
      canvas.drawLine(box.x + size / 2, box.y + size - 3, box.x + size - 3, box.y + 3, mark);
    }
    if (this.label) {
      const font = new Skia.Font(null, 14);
      const textPaint = new Skia.Paint(); textPaint.setColor(Skia.Color4f(0, 0, 0, 1));
      const blob = Skia.TextBlob.MakeFromText(this.label, font);
      canvas.drawTextBlob(blob, this.x + 6, this.y + (this.h + this.fontSize) / 2 - 2, textPaint);
    }
  }
  _handleEvent(ev) {
    if (ev.type === 'pointerup' && this._containsPoint(ev.x, ev.y)) {
      this.checked = !this.checked;
      if (this.onChange) this.onChange(this.checked);
      this.emit('change', this.checked);
      return true;
    }
    return false;
  }
}

export class TextInput extends Widget {
  constructor(text = '', onCommit = null, props = {}) {
    super(props);
    this.text = text;
    this.onCommit = onCommit;
    this._focused = false;
  }
  _draw(canvas, Skia) {
    const paint = new Skia.Paint(); paint.setAntiAlias(true);
    paint.setColor(Skia.Color4f(1, 1, 1, 1));
    canvas.drawRect({ x: this.x, y: this.y, width: this.w, height: this.h }, paint);
    const border = new Skia.Paint(); border.setStyle(Skia.PaintStyle.Stroke); border.setStrokeWidth(1);
    border.setColor(Skia.Color4f(0.3, 0.3, 0.3, 1));
    canvas.drawRect({ x: this.x, y: this.y, width: this.w, height: this.h }, border);
    const font = new Skia.Font(null, 14);
    const tpaint = new Skia.Paint(); tpaint.setColor(Skia.Color4f(0, 0, 0, 1));
    const blob = Skia.TextBlob.MakeFromText(this.text, font);
    canvas.drawTextBlob(blob, this.x + 6, this.y + (this.h + this.fontSize) / 2 - 2, tpaint);
  }
  _handleEvent(ev) {
    if (ev.type === 'pointerup' && this._containsPoint(ev.x, ev.y)) { this._focused = true; return true; }
    if (ev.type === 'pointerup') this._focused = false;
    return false;
  }
}

(async function example() {
  const canvas = document.getElementById('skiaCanvas');
  const Skia = await loadSkia();
  const gui = new SkiaGUI(canvas, Skia);
  await gui.init();

  robotoData = await (await fetch('https://cdn.skia.org/misc/Roboto-Regular.ttf')).arrayBuffer();

  const root = new HBox({}, [
    new VBox({ spacing: 10, x: 64, y: 64 }, [
      new Button('Increase', () => console.log('test !'), { w: 100, h: 100 }),
      new Button('Decrease', () => console.log('test !'), { w: 100, h: 100 }),
    ])
  ]);
  gui.setRoot(root);
})();
