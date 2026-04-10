/**
 * Arkheion — Barcode Scanner wrapper
 *
 * Strategy:
 *   1. Native BarcodeDetector (Chrome/Edge on Android) — fast, hardware accelerated
 *   2. ZXing BrowserMultiFormatReader.decodeFromStream() (all other browsers inc. iOS Safari)
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

    // Check which strategy we'll use BEFORE touching the video element
    const nativeOk = await this._nativeSupportsEAN();

    const constraints = {
      video: deviceId
        ? { deviceId: { exact: deviceId }, width: { ideal: 1280 }, height: { ideal: 720 } }
        : { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } }
    };

    // Get the camera stream
    this._stream = await navigator.mediaDevices.getUserMedia(constraints);
    this.scanning = true;

    // Both paths: WE manage the video element (set srcObject and play).
    // This must happen synchronously within user-gesture context to satisfy
    // iOS Safari's autoplay policy. ZXing's decodeFromStream has an internal
    // playVideoOnLoadAsync() that deadlocks, so we bypass it entirely.
    this.videoElement.srcObject = this._stream;
    this.videoElement.setAttribute('playsinline', '');
    await this.videoElement.play();

    if (nativeOk) {
      this._startNative();
    } else {
      await this._startZXing();
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

  /** Native BarcodeDetector (Chrome/Brave/Edge) — fast, hardware accelerated */
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
   * ZXing BrowserMultiFormatReader — works on ALL browsers.
   *
   * IMPORTANT: We do NOT use decodeFromStream / decodeFromVideoDevice because
   * ZXing's internal playVideoOnLoadAsync() deadlocks on iOS Safari — it waits
   * for a 'canplay' event that may have already fired or never fires.
   *
   * Instead we:
   *   1. Manage the video ourselves (srcObject + play) — done in start() before
   *      this method is called, within user-gesture context.
   *   2. Use a requestAnimationFrame loop to call reader.decode(videoElement)
   *      which captures the current frame to canvas and decodes it.
   */
  async _startZXing() {
    const zxing = await import('@zxing/library');

    const hints = new Map();
    hints.set(zxing.DecodeHintType.POSSIBLE_FORMATS, [
      zxing.BarcodeFormat.EAN_13,
      zxing.BarcodeFormat.EAN_8,
      zxing.BarcodeFormat.CODE_128,
      zxing.BarcodeFormat.CODE_39,
      zxing.BarcodeFormat.UPC_A,
      zxing.BarcodeFormat.UPC_E
    ]);
    hints.set(zxing.DecodeHintType.TRY_HARDER, true);

    const reader = new zxing.BrowserMultiFormatReader(hints, 200);
    this._zxingReader = reader;

    // Frame-by-frame decode loop using requestAnimationFrame
    let frameCount = 0;
    const tick = () => {
      if (!this.scanning) return;

      if (this.videoElement.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
        frameCount++;
        // Decode every 3rd frame (~20fps on 60fps display) to save CPU
        if (frameCount % 3 === 0) {
          try {
            // reader.decode() draws video frame to internal canvas, creates
            // HTMLCanvasElementLuminanceSource, and runs the decode pipeline.
            const result = reader.decode(this.videoElement);
            if (result) {
              this._emitResult(result.getText());
            }
          } catch (e) {
            // NotFoundException is thrown every frame without a barcode — ignore
            if (e.name !== 'NotFoundException' &&
                e.constructor?.name !== 'NotFoundException' &&
                !e.message?.includes('No MultiFormat')) {
              console.warn('[Arkheion scanner] decode error:', e.message);
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
        this._zxingReader.stopContinuousDecode();
        this._zxingReader.reset();
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
