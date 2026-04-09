/**
 * Arkheion — ZXing Barcode Scanner wrapper
 *
 * Wraps @zxing/library to provide a simple start/stop API for scanning
 * ISBN barcodes from the device camera. Import this only in browser context.
 *
 * Usage (inside onMount or browser-only code):
 *   const { BarcodeScanner } = await import('$lib/scanner.js');
 *   const scanner = new BarcodeScanner(videoElement);
 *   scanner.onResult = (isbn) => console.log('Scanned:', isbn);
 *   scanner.onError = (err) => console.error(err);
 *   await scanner.start();
 *   // ...
 *   scanner.stop();
 */

import {
  BrowserMultiFormatReader,
  DecodeHintType,
  BarcodeFormat
} from '@zxing/library';

export class BarcodeScanner {
  constructor(videoElement) {
    this.videoElement = videoElement;
    this.reader = null;
    this.scanning = false;
    this.lastResult = null;
    this.debounceMs = 1500;
    this.lastScanTime = 0;

    /** @type {(isbn: string) => void} */
    this.onResult = null;
    /** @type {(error: Error) => void} */
    this.onError = null;
  }

  /**
   * Start the camera and begin scanning for barcodes.
   * @param {string} [deviceId] - Optional camera device ID.
   */
  async start(deviceId = null) {
    if (this.scanning) return;

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

    this.reader = new BrowserMultiFormatReader(hints);
    this.scanning = true;

    try {
      const selectedDeviceId = deviceId ?? await this._getPreferredCameraId();

      await this.reader.decodeFromVideoDevice(
        selectedDeviceId,
        this.videoElement,
        (result, error) => {
          if (result) {
            const now = Date.now();
            if (now - this.lastScanTime < this.debounceMs) return;
            if (result.text === this.lastResult) return;

            this.lastResult = result.text;
            this.lastScanTime = now;

            if (this.onResult) {
              this.onResult(result.text);
            }
          }
          if (error && error.name !== 'NotFoundException') {
            if (this.onError) {
              this.onError(error);
            }
          }
        }
      );
    } catch (err) {
      this.scanning = false;
      if (this.onError) {
        this.onError(err);
      }
      throw err;
    }
  }

  /**
   * Stop the scanner and release the camera.
   */
  stop() {
    if (this.reader) {
      this.reader.reset();
      this.reader = null;
    }
    this.scanning = false;
    this.lastResult = null;
  }

  /**
   * List available video input devices.
   * @returns {Promise<MediaDeviceInfo[]>}
   */
  static async listCameras() {
    return BrowserMultiFormatReader.listVideoInputDevices();
  }

  /**
   * Returns the device ID of the preferred camera.
   * On mobile, prefers rear camera. On desktop, uses default (null = first available).
   * @private
   */
  async _getPreferredCameraId() {
    try {
      const devices = await BarcodeScanner.listCameras();
      if (!devices.length) return null;

      // On mobile, prefer rear/environment camera
      const rear = devices.find(
        (d) =>
          d.label.toLowerCase().includes('back') ||
          d.label.toLowerCase().includes('environment') ||
          d.label.toLowerCase().includes('rear')
      );
      // On desktop or if no rear camera found, use null (ZXing picks the default/first)
      return rear?.deviceId ?? null;
    } catch {
      return null;
    }
  }
}

/**
 * Check if the browser supports camera access via getUserMedia.
 * @returns {boolean}
 */
export function isCameraSupported() {
  return typeof navigator !== 'undefined' &&
    !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
}
