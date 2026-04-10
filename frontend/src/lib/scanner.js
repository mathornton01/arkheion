/**
 * Arkheion -- Barcode Scanner wrapper
 *
 * Strategy:
 *   1. Native BarcodeDetector (Chrome/Edge on Android) -- fast, hardware accelerated
 *   2. ZXing BrowserMultiFormatReader.decodeFromVideoDevice() (all other browsers)
 *
 * Usage:
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
    this._zxingReader = null;
    this._scanCanvas = null;
    this._scanCtx = null;

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

    const nativeOk = await this._nativeSupportsEAN();
    this.scanning = true;

    if (nativeOk) {
      // Native path: we manage the video ourselves
      const constraints = {
        video: deviceId
          ? { deviceId: { exact: deviceId }, width: { ideal: 1280 }, height: { ideal: 720 } }
          : { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } }
      };

      this._stream = await navigator.mediaDevices.getUserMedia(constraints);
      this.videoElement.srcObject = this._stream;
      this.videoElement.setAttribute('playsinline', '');
      await this.videoElement.play();
      this._startNative();
    } else {
      // ZXing path: we manage video + use canvas-based decode
      await this._startZXing(deviceId);
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
            this._emitResult(raw);
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
   * ZXing BrowserMultiFormatReader -- works on ALL browsers.
   *
   * We manage the video ourselves, draw frames to a hidden canvas,
   * then use ZXing's low-level decode pipeline (HTMLCanvasElementLuminanceSource
   * -> HybridBinarizer -> BinaryBitmap -> MultiFormatReader.decode).
   *
   * This avoids ZXing's playVideoOnLoadAsync() which deadlocks on iOS Safari.
   */
  async _startZXing(deviceId) {
    const zxing = await import('@zxing/library');

    // Get camera stream ourselves
    const constraints = {
      video: deviceId
        ? { deviceId: { exact: deviceId }, width: { ideal: 1280 }, height: { ideal: 720 } }
        : { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } }
    };

    this._stream = await navigator.mediaDevices.getUserMedia(constraints);
    this.videoElement.srcObject = this._stream;
    this.videoElement.setAttribute('playsinline', '');
    await this.videoElement.play();

    // Configure formats
    const formats = [
      zxing.BarcodeFormat.EAN_13,
      zxing.BarcodeFormat.EAN_8,
      zxing.BarcodeFormat.CODE_128,
      zxing.BarcodeFormat.CODE_39,
      zxing.BarcodeFormat.UPC_A,
      zxing.BarcodeFormat.UPC_E
    ];

    const hints = new Map();
    hints.set(zxing.DecodeHintType.POSSIBLE_FORMATS, formats);
    hints.set(zxing.DecodeHintType.TRY_HARDER, true);

    // Create the underlying multi-format reader (NOT BrowserMultiFormatReader)
    const reader = new zxing.MultiFormatReader();
    reader.setHints(hints);
    this._zxingReader = reader;

    // Hidden canvas for frame capture
    this._scanCanvas = document.createElement('canvas');
    this._scanCtx = this._scanCanvas.getContext('2d', { willReadFrequently: true });

    // Store ZXing classes we need for decode loop
    const HTMLCanvasLuminance = zxing.HTMLCanvasElementLuminanceSource;
    const HybridBinarizer = zxing.HybridBinarizer;
    const BinaryBitmap = zxing.BinaryBitmap;
    const NotFoundException = zxing.NotFoundException;

    console.log('[Arkheion scanner] ZXing initialized, starting decode loop');
    console.log('[Arkheion scanner] HTMLCanvasElementLuminanceSource available:', !!HTMLCanvasLuminance);

    let frameCount = 0;
    let lastDecodeTime = 0;
    const DECODE_INTERVAL_MS = 150; // decode every 150ms (~6-7 fps)

    const tick = (timestamp) => {
      if (!this.scanning) return;

      if (this.videoElement.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
        // Throttle by time instead of frame count for consistency
        if (timestamp - lastDecodeTime >= DECODE_INTERVAL_MS) {
          lastDecodeTime = timestamp;
          frameCount++;

          const vw = this.videoElement.videoWidth;
          const vh = this.videoElement.videoHeight;

          if (vw > 0 && vh > 0) {
            // Update canvas size if needed
            if (this._scanCanvas.width !== vw || this._scanCanvas.height !== vh) {
              this._scanCanvas.width = vw;
              this._scanCanvas.height = vh;
              console.log(`[Arkheion scanner] Canvas size: ${vw}x${vh}`);
            }

            // Draw current video frame to canvas
            this._scanCtx.drawImage(this.videoElement, 0, 0, vw, vh);

            try {
              let result = null;

              if (HTMLCanvasLuminance) {
                // Preferred: use ZXing's canvas luminance source
                const luminanceSource = new HTMLCanvasLuminance(this._scanCanvas);
                const binaryBitmap = new BinaryBitmap(new HybridBinarizer(luminanceSource));
                result = reader.decode(binaryBitmap);
              } else {
                // Fallback: manually extract image data and create luminance source
                const imageData = this._scanCtx.getImageData(0, 0, vw, vh);
                const luminances = new Uint8ClampedArray(vw * vh);
                for (let i = 0; i < vw * vh; i++) {
                  const r = imageData.data[i * 4];
                  const g = imageData.data[i * 4 + 1];
                  const b = imageData.data[i * 4 + 2];
                  // ITU-R BT.601 luma
                  luminances[i] = (r * 0.299 + g * 0.587 + b * 0.114) | 0;
                }
                const luminanceSource = new zxing.RGBLuminanceSource(luminances, vw, vh);
                const binaryBitmap = new BinaryBitmap(new HybridBinarizer(luminanceSource));
                result = reader.decode(binaryBitmap);
              }

              if (result) {
                const text = result.getText();
                console.log('[Arkheion scanner] DECODED:', text);
                this._emitResult(text);
              }
            } catch (e) {
              // NotFoundException fires on every frame without a barcode -- normal
              if (e instanceof NotFoundException ||
                  e.name === 'NotFoundException' ||
                  e.constructor?.name === 'NotFoundException') {
                // Normal -- no barcode in this frame
                if (frameCount % 30 === 0) {
                  console.log(`[Arkheion scanner] Scanning... (${frameCount} frames processed)`);
                }
              } else {
                console.warn('[Arkheion scanner] decode error:', e.name, e.message);
              }
            }
          }
        }
      }

      if (this.scanning) {
        this._animFrameId = requestAnimationFrame(tick);
      }
    };

    this._animFrameId = requestAnimationFrame(tick);
  }

  _emitResult(text) {
    if (!text) return;
    const now = Date.now();
    if (text !== this.lastResult || now - this.lastScanTime >= this.debounceMs) {
      this.lastResult = text;
      this.lastScanTime = now;
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

    if (this._zxingReader) {
      try {
        if (this._zxingReader.reset) this._zxingReader.reset();
      } catch { /* ignore */ }
      this._zxingReader = null;
    }

    if (this._stream) {
      this._stream.getTracks().forEach(t => t.stop());
      this._stream = null;
    }

    if (this.videoElement) {
      this.videoElement.srcObject = null;
    }

    this._scanCanvas = null;
    this._scanCtx = null;
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
