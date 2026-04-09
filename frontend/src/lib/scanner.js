/**
 * Arkheion — ZXing Barcode Scanner wrapper
 *
 * Wraps @zxing/library to provide a simple start/stop API for scanning
 * ISBN barcodes from the device camera. Designed for use in the /scan route.
 *
 * Usage:
 *   import { BarcodeScanner } from '$lib/scanner.js';
 *
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
    this.debounceMs = 1500; // Prevent duplicate scans within 1.5s
    this.lastScanTime = 0;

    /** @type {(isbn: string) => void} */
    this.onResult = null;
    /** @type {(error: Error) => void} */
    this.onError = null;
  }

  /**
   * Start the camera and begin scanning for barcodes.
   * Prompts for camera permission if not already granted.
   * @param {string} [deviceId] - Optional camera device ID. Uses environment-facing camera by default.
   */
  async start(deviceId = null) {
    if (this.scanning) return;

    // Configure ZXing to look for 1D barcode formats (ISBN barcodes are EAN-13)
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
      // If no specific deviceId, pick the environment-facing (rear) camera
      const selectedDeviceId = deviceId ?? await this._getRearCameraId();

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
            // NotFoundException is normal (no barcode in frame) — ignore it
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
    const devices = await BrowserMultiFormatReader.listVideoInputDevices();
    return devices;
  }

  /**
   * Returns the device ID of the environment-facing (rear) camera,
   * or null to let ZXing choose the default.
   * @private
   */
  async _getRearCameraId() {
    try {
      const devices = await BarcodeScanner.listCameras();
      // Prefer "environment" or "back" in the label
      const rear = devices.find(
        (d) =>
          d.label.toLowerCase().includes('back') ||
          d.label.toLowerCase().includes('environment') ||
          d.label.toLowerCase().includes('rear')
      );
      return rear?.deviceId ?? null;
    } catch {
      return null;
    }
  }
}

/**
 * Check if the browser supports camera access.
 * @returns {boolean}
 */
export function isCameraSupported() {
  return !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
}

/**
 * Request camera permission proactively (shows browser permission dialog).
 * @returns {Promise<boolean>} true if permission granted
 */
export async function requestCameraPermission() {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ video: true });
    // Release the stream immediately — we just want to trigger the permission dialog
    stream.getTracks().forEach((t) => t.stop());
    return true;
  } catch {
    return false;
  }
}
