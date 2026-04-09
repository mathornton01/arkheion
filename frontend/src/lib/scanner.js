/**
 * Arkheion — Barcode Scanner wrapper
 *
 * Uses the native BarcodeDetector API (Chrome/Brave/Edge) as primary scanner,
 * falls back to @zxing/library for Firefox and other browsers.
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

    // Acquire camera stream ourselves so we control it
    const constraints = {
      video: deviceId
        ? { deviceId: { exact: deviceId }, width: { ideal: 1280 }, height: { ideal: 720 } }
        : { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } }
    };

    this._stream = await navigator.mediaDevices.getUserMedia(constraints);
    this.videoElement.srcObject = this._stream;
    this.videoElement.setAttribute('playsinline', '');
    await this.videoElement.play();
    this.scanning = true;

    // Check if native BarcodeDetector actually supports EAN-13
    // On Linux Chrome/Brave, BarcodeDetector may exist but lack EAN support
    const nativeOk = await this._nativeSupportsEAN();
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

  /** Native BarcodeDetector (Chrome/Brave/Edge) - fast, hardware accelerated */
  _startNative() {
    let detector;
    try {
      // 'isbn' is not a valid BarcodeDetector format — omit it
      detector = new BarcodeDetector({
        formats: ['ean_13', 'ean_8', 'upc_a', 'upc_e', 'code_128', 'code_39', 'qr_code']
      });
    } catch {
      // formats list not supported, try without
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
          // Detection errors on a single frame are normal — keep going
        }
      }

      if (this.scanning) {
        this._animFrameId = requestAnimationFrame(tick);
      }
    };

    this._animFrameId = requestAnimationFrame(tick);
  }

  /** ZXing fallback for Firefox etc. */
  async _startZXing() {
    const { BrowserMultiFormatReader, DecodeHintType, BarcodeFormat } =
      await import('@zxing/library');

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
    hints.set(DecodeHintType.ALSO_INVERTED, true);

    // Scan at 400ms intervals for more decode attempts per second
    this._zxingReader = new BrowserMultiFormatReader(hints, { delayBetweenScanAttempts: 200 });

    // Pass our existing stream directly so ZXing doesn't try to open the camera
    // again (decodeFromVideoDevice(null) would conflict with our already-open stream).
    await this._zxingReader.decodeFromStream(this._stream, this.videoElement, (result, error) => {
      if (result) {
        const now = Date.now();
        if (result.text !== this.lastResult || now - this.lastScanTime >= this.debounceMs) {
          this.lastResult = result.text;
          this.lastScanTime = now;
          if (this.onResult) this.onResult(result.text);
        }
      }
      if (error) {
        // NotFoundException fires every frame when no barcode is visible — suppress it
        const msg = error.message || '';
        const isNotFound =
          error.name === 'NotFoundException' ||
          msg.includes('No MultiFormat') ||
          msg.includes('not found') ||
          msg.includes('2D') ||
          msg === '';
        if (!isNotFound && this.onError) {
          this.onError(error);
        }
      }
    });
  }

  /** Stop scanning and release camera. */
  stop() {
    this.scanning = false;

    if (this._animFrameId) {
      cancelAnimationFrame(this._animFrameId);
      this._animFrameId = null;
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
    const { BrowserMultiFormatReader } = await import('@zxing/library');
    return BrowserMultiFormatReader.listVideoInputDevices();
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
