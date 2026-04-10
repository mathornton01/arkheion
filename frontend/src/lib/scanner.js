/**
 * Arkheion -- Barcode Scanner wrapper (v2 -- improved webcam support)
 *
 * Strategy:
 *   1. Native BarcodeDetector (Chrome/Edge) -- fast, hardware accelerated
 *   2. Canvas frame-capture + ZXing MultiFormatReader -- all other browsers
 *
 * Webcam improvements over v1:
 *   - Multi-scale scanning: center crop, 2x zoomed crop, full frame
 *   - Dual binarizer per attempt: HybridBinarizer -> GlobalHistogramBinarizer
 *   - Contrast boost applied to canvas before decode (ctx.filter)
 *   - Inverted luminance fallback (white bars on dark background)
 *   - Faster scan interval (120ms)
 *   - Better camera constraints (720p @ 30fps, focusMode continuous)
 *
 * Usage (inside onMount or browser-only code):
 *   const { BarcodeScanner } = await import('$lib/scanner.js');
 *   const scanner = new BarcodeScanner(videoElement);
 *   scanner.onResult = (isbn) => console.log('Scanned:', isbn);
 *   scanner.onError = (err) => console.error(err);
 *   await scanner.start();
 *   scanner.stop();
 */

export class BarcodeScanner {
  constructor(videoElement) {
    this.videoElement = videoElement;
    this.scanning = false;
    this.lastResult = null;
    this.debounceMs = 1500;
    this.lastScanTime = 0;
    this._stream = null;
    this._animFrameId = null;
    this._scanTimeout = null;
    this._zxingReader = null;

    /** @type {(isbn: string) => void} */
    this.onResult = null;
    /** @type {(error: Error) => void} */
    this.onError = null;
  }

  /**
   * Start camera and scanning.
   * @param {string|null} [deviceId]
   */
  async start(deviceId = null) {
    if (this.scanning) return;

    const constraints = {
      video: deviceId
        ? {
            deviceId: { exact: deviceId },
            width: { ideal: 1280 },
            height: { ideal: 720 },
            frameRate: { ideal: 30, max: 30 }
          }
        : {
            facingMode: { ideal: 'environment' },
            width: { ideal: 1280 },
            height: { ideal: 720 },
            frameRate: { ideal: 30, max: 30 }
          }
    };

    this._stream = await navigator.mediaDevices.getUserMedia(constraints);
    this.videoElement.srcObject = this._stream;
    this.videoElement.setAttribute('playsinline', '');
    await this.videoElement.play();
    this.scanning = true;

    // Try to enable continuous autofocus on webcams that support it
    try {
      const track = this._stream.getVideoTracks()[0];
      const caps = track.getCapabilities?.() || {};
      if (caps.focusMode && caps.focusMode.includes('continuous')) {
        await track.applyConstraints({ advanced: [{ focusMode: 'continuous' }] });
        console.log('[Scanner] Continuous autofocus enabled');
      }
    } catch {
      // Not supported -- silently ignore
    }

    const nativeOk = await this._nativeSupportsEAN();
    if (nativeOk) {
      console.log('[Scanner] Using native BarcodeDetector');
      this._startNative();
    } else {
      console.log('[Scanner] Using ZXing canvas fallback (multi-scale + dual binarizer)');
      await this._startZXingCanvas();
    }
  }

  /** Returns true if BarcodeDetector is available and supports EAN-13 */
  async _nativeSupportsEAN() {
    if (typeof BarcodeDetector === 'undefined') return false;
    try {
      const supported = await BarcodeDetector.getSupportedFormats();
      return supported.includes('ean_13') || supported.includes('ean_8');
    } catch {
      return false;
    }
  }

  /** Native BarcodeDetector (Chrome/Brave/Edge) -- fast, hardware accelerated */
  _startNative() {
    let detector;
    try {
      detector = new BarcodeDetector({
        formats: ['ean_13', 'ean_8', 'upc_a', 'upc_e', 'code_128', 'code_39', 'qr_code']
      });
    } catch {
      try {
        detector = new BarcodeDetector();
      } catch (e) {
        if (this.onError) this.onError(e);
        return;
      }
    }

    const tick = async () => {
      if (!this.scanning) return;

      if (this.videoElement.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
        try {
          const codes = await detector.detect(this.videoElement);
          if (codes.length > 0) {
            const raw = codes[0].rawValue;
            const now = Date.now();
            if (raw !== this.lastResult || now - this.lastScanTime >= this.debounceMs) {
              this.lastResult = raw;
              this.lastScanTime = now;
              if (this.onResult) this.onResult(raw);
            }
          }
        } catch {
          // Detection errors on a single frame are normal
        }
      }

      if (this.scanning) {
        this._animFrameId = requestAnimationFrame(tick);
      }
    };

    this._animFrameId = requestAnimationFrame(tick);
  }

  /**
   * Canvas-based ZXing fallback -- improved for desktop webcams.
   *
   * Per scan cycle (every ~120ms):
   *   Pass 1: Center crop (70%x40%) -- HybridBinarizer with contrast boost
   *   Pass 2: Center crop -- GlobalHistogramBinarizer (different failure modes)
   *   Pass 3: 2x zoomed center crop (35%x20%) -- HybridBinarizer (barcode far from cam)
   *   Pass 4: Every 4th frame -- full frame with HybridBinarizer
   *   Pass 5: Every 8th frame -- center crop with inverted luminance (dark bg barcodes)
   */
  async _startZXingCanvas() {
    let zxing;
    try {
      zxing = await import('@zxing/library');
    } catch (e) {
      console.error('[Scanner] Failed to load @zxing/library:', e);
      if (this.onError) this.onError(e);
      return;
    }

    const {
      MultiFormatReader,
      DecodeHintType,
      BarcodeFormat,
      BinaryBitmap,
      HybridBinarizer,
      GlobalHistogramBinarizer,
      HTMLCanvasElementLuminanceSource,
      InvertedLuminanceSource
    } = zxing;

    if (!HTMLCanvasElementLuminanceSource) {
      const err = new Error('HTMLCanvasElementLuminanceSource not available in @zxing/library');
      console.error('[Scanner]', err.message);
      if (this.onError) this.onError(err);
      return;
    }

    const hints = new Map();
    hints.set(DecodeHintType.POSSIBLE_FORMATS, [
      BarcodeFormat.EAN_13,
      BarcodeFormat.EAN_8,
      BarcodeFormat.CODE_128,
      BarcodeFormat.CODE_39,
      BarcodeFormat.UPC_A,
      BarcodeFormat.UPC_E
    ]);
    hints.set(DecodeHintType.TRY_HARDER, true);

    const reader = new MultiFormatReader();
    reader.setHints(hints);
    this._zxingReader = reader;

    // Offscreen canvases -- do NOT pass willReadFrequently, HTMLCanvasElementLuminanceSource
    // calls canvas.getContext('2d') internally and mismatched options cause issues on Safari.
    const cropCanvas = document.createElement('canvas');
    const fullCanvas = document.createElement('canvas');

    let frameCount = 0;
    let errorCount = 0;

    /**
     * Try to decode a canvas with multiple binarizer strategies.
     * Returns true if a barcode was found.
     * @param {HTMLCanvasElement} canvas
     * @param {boolean} [tryInverted]
     */
    const tryDecode = (canvas, tryInverted = false) => {
      const lum = new HTMLCanvasElementLuminanceSource(canvas);

      // Pass A: HybridBinarizer (handles uneven lighting / shadows)
      try {
        const result = reader.decode(new BinaryBitmap(new HybridBinarizer(lum)));
        if (result) { this._handleZXingResult(result.getText()); return true; }
      } catch (e) {
        if (e.constructor?.name !== 'NotFoundException') {
          errorCount++;
          if (errorCount <= 5) console.warn('[Scanner] HybridBinarizer error:', e.constructor?.name);
        }
      }

      // Pass B: GlobalHistogramBinarizer (better for evenly-lit high-contrast images)
      if (GlobalHistogramBinarizer) {
        try {
          const result = reader.decode(new BinaryBitmap(new GlobalHistogramBinarizer(lum)));
          if (result) { this._handleZXingResult(result.getText()); return true; }
        } catch (e) {
          if (e.constructor?.name !== 'NotFoundException') {
            errorCount++;
            if (errorCount <= 5) console.warn('[Scanner] GlobalHistogramBinarizer error:', e.constructor?.name);
          }
        }
      }

      // Pass C: Inverted luminance (white bars on dark background)
      if (tryInverted && InvertedLuminanceSource) {
        try {
          const invLum = new InvertedLuminanceSource(lum);
          const result = reader.decode(new BinaryBitmap(new HybridBinarizer(invLum)));
          if (result) { this._handleZXingResult(result.getText()); return true; }
        } catch {
          // Expected
        }
      }

      return false;
    };

    /**
     * Draw a region of the video to a canvas with a contrast boost filter.
     */
    const drawWithBoost = (canvas, sx, sy, sw, sh, dw, dh) => {
      canvas.width = dw || sw;
      canvas.height = dh || sh;
      const ctx = canvas.getContext('2d');
      // Contrast + brightness boost: helps with dim webcam images
      ctx.filter = 'contrast(1.4) brightness(1.1)';
      ctx.drawImage(this.videoElement, sx, sy, sw, sh, 0, 0, dw || sw, dh || sh);
      ctx.filter = 'none';
    };

    console.log('[Scanner] ZXing multi-scale scan loop starting');

    const scanFrame = () => {
      if (!this.scanning) return;

      const vw = this.videoElement.videoWidth;
      const vh = this.videoElement.videoHeight;

      if (vw && vh && this.videoElement.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
        let found = false;

        // === Pass 1: Center crop (70%x40%) -- where the scan overlay is ===
        const cropW = Math.round(vw * 0.7);
        const cropH = Math.round(vh * 0.4);
        const cropX = Math.round((vw - cropW) / 2);
        const cropY = Math.round((vh - cropH) / 2);

        try {
          drawWithBoost(cropCanvas, cropX, cropY, cropW, cropH);
          found = tryDecode(cropCanvas);
        } catch (e) {
          console.warn('[Scanner] Pass 1 draw error:', e.message);
        }

        // === Pass 2: 2x zoomed center crop (35%x20%) -- barcode held farther away ===
        if (!found) {
          const zoom2W = Math.round(vw * 0.35);
          const zoom2H = Math.round(vh * 0.2);
          const zoom2X = Math.round((vw - zoom2W) / 2);
          const zoom2Y = Math.round((vh - zoom2H) / 2);

          try {
            // Scale the smaller region UP = effective 2x software zoom
            drawWithBoost(cropCanvas, zoom2X, zoom2Y, zoom2W, zoom2H, zoom2W * 2, zoom2H * 2);
            found = tryDecode(cropCanvas);
          } catch (e) {
            console.warn('[Scanner] Pass 2 draw error:', e.message);
          }
        }

        // === Pass 3: Full frame (every 4th frame) -- barcode at edge or close up ===
        if (!found && frameCount % 4 === 0) {
          try {
            drawWithBoost(fullCanvas, 0, 0, vw, vh);
            found = tryDecode(fullCanvas);
          } catch (e) {
            console.warn('[Scanner] Pass 3 draw error:', e.message);
          }
        }

        // === Pass 4: Inverted center crop (every 8th frame) -- dark background labels ===
        if (!found && frameCount % 8 === 0 && InvertedLuminanceSource) {
          try {
            drawWithBoost(cropCanvas, cropX, cropY, cropW, cropH);
            tryDecode(cropCanvas, true);
          } catch {
            // Silently ignore
          }
        }

        frameCount++;
        if (frameCount === 1) {
          console.log(`[Scanner] First frame: ${vw}x${vh}, passes: center -> 2x zoom -> full -> inverted`);
        }
      }

      this._scanTimeout = setTimeout(scanFrame, 120);
    };

    // Small delay to let the video stabilize before first scan
    this._scanTimeout = setTimeout(scanFrame, 300);
  }

  _handleZXingResult(text) {
    if (!text) return;
    const now = Date.now();
    if (text !== this.lastResult || now - this.lastScanTime >= this.debounceMs) {
      this.lastResult = text;
      this.lastScanTime = now;
      console.log('[Scanner] Barcode detected:', text);
      if (this.onResult) this.onResult(text);
    }
  }

  /** Stop scanning and release camera. */
  stop() {
    this.scanning = false;

    if (this._animFrameId) {
      cancelAnimationFrame(this._animFrameId);
      this._animFrameId = null;
    }

    if (this._scanTimeout) {
      clearTimeout(this._scanTimeout);
      this._scanTimeout = null;
    }

    if (this._zxingReader) {
      try { this._zxingReader.reset(); } catch { /* ignore */ }
      this._zxingReader = null;
    }

    if (this._stream) {
      this._stream.getTracks().forEach(t => t.stop());
      this._stream = null;
    }

    if (this.videoElement) {
      this.videoElement.srcObject = null;
    }

    this.lastResult = null;
  }

  /** List available video input devices. */
  static async listCameras() {
    const devices = await navigator.mediaDevices.enumerateDevices();
    return devices.filter(d => d.kind === 'videoinput');
  }
}

/**
 * Check if the browser supports camera access via getUserMedia.
 * @returns {boolean}
 */
export function isCameraSupported() {
  return (
    typeof navigator !== 'undefined' &&
    !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia)
  );
}

/**
 * Request camera permission proactively (shows browser permission dialog).
 * @returns {Promise<boolean>} true if permission granted
 */
export async function requestCameraPermission() {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ video: true });
    stream.getTracks().forEach((t) => t.stop());
    return true;
  } catch {
    return false;
  }
}
