// CGWindowListCreateImage is marked API_UNAVAILABLE in the macOS 15 SDK
// in favor of ScreenCaptureKit, but it still works at runtime. We load it
// dynamically via dlsym to bypass the SDK availability annotation.
// TODO: migrate to ScreenCaptureKit.
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <ImageIO/ImageIO.h>
#import <AppKit/AppKit.h>
#include <dlfcn.h>
#include <stdlib.h>
#include "screenshot.h"

// Function pointer type matching CGWindowListCreateImage signature.
typedef CGImageRef (*CGWindowListCreateImageFunc)(CGRect, CGWindowListOption,
    CGWindowID, CGWindowImageOption);

// Lazily resolve CGWindowListCreateImage at runtime via dlsym.
static CGWindowListCreateImageFunc get_capture_func(void) {
    static CGWindowListCreateImageFunc fn = NULL;
    static int resolved = 0;
    if (!resolved) {
        fn = (CGWindowListCreateImageFunc)dlsym(RTLD_DEFAULT,
            "CGWindowListCreateImage");
        resolved = 1;
    }
    return fn;
}

// Encode a CGImage to PNG or JPEG data.
// format: 0=PNG, 1=JPEG
// quality: 1-100 (only for JPEG)
// Returns 0 on success, -1 on failure.
static int encode_image(CGImageRef image, int format, int quality,
                        unsigned char** outData, int* outLength) {
    CFStringRef type;
    if (format == 1) {
        type = CFSTR("public.jpeg");
    } else {
        type = CFSTR("public.png");
    }

    CFMutableDataRef data = CFDataCreateMutable(kCFAllocatorDefault, 0);
    if (!data) return -1;

    CGImageDestinationRef dest = CGImageDestinationCreateWithData(data, type, 1, NULL);
    if (!dest) {
        CFRelease(data);
        return -1;
    }

    if (format == 1) {
        float q = (float)quality / 100.0f;
        CFNumberRef qualityNum = CFNumberCreate(kCFAllocatorDefault,
            kCFNumberFloatType, &q);
        CFStringRef keys[] = { kCGImageDestinationLossyCompressionQuality };
        CFTypeRef values[] = { qualityNum };
        CFDictionaryRef props = CFDictionaryCreate(kCFAllocatorDefault,
            (const void**)keys, (const void**)values, 1,
            &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
        CGImageDestinationAddImage(dest, image, props);
        CFRelease(props);
        CFRelease(qualityNum);
    } else {
        CGImageDestinationAddImage(dest, image, NULL);
    }

    if (!CGImageDestinationFinalize(dest)) {
        CFRelease(dest);
        CFRelease(data);
        return -1;
    }
    CFRelease(dest);

    CFIndex len = CFDataGetLength(data);
    unsigned char* buf = (unsigned char*)malloc(len);
    if (!buf) {
        CFRelease(data);
        return -1;
    }
    CFDataGetBytes(data, CFRangeMake(0, len), buf);
    CFRelease(data);

    *outData = buf;
    *outLength = (int)len;
    return 0;
}

// Scale a CGImage down by the given factor.
// Returns a new CGImageRef that the caller must release, or NULL on failure.
static CGImageRef scale_image(CGImageRef image, float scale) {
    if (scale >= 1.0f) return NULL;

    size_t origW = CGImageGetWidth(image);
    size_t origH = CGImageGetHeight(image);
    size_t newW = (size_t)(origW * scale);
    size_t newH = (size_t)(origH * scale);
    if (newW == 0) newW = 1;
    if (newH == 0) newH = 1;

    CGColorSpaceRef colorSpace = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
    if (!colorSpace) return NULL;

    CGContextRef ctx = CGBitmapContextCreate(NULL, newW, newH, 8, 0, colorSpace,
        (CGBitmapInfo)kCGImageAlphaPremultipliedLast);
    CGColorSpaceRelease(colorSpace);
    if (!ctx) return NULL;

    CGContextSetInterpolationQuality(ctx, kCGInterpolationHigh);
    CGContextDrawImage(ctx, CGRectMake(0, 0, newW, newH), image);

    CGImageRef scaled = CGBitmapContextCreateImage(ctx);
    CGContextRelease(ctx);
    return scaled;
}

int cg_capture_window(int windowID, int format, int quality, float scale,
                      ScreenshotResult* result) {
    CGWindowListCreateImageFunc captureFn = get_capture_func();
    if (!captureFn) return -1;

    CGImageRef image = captureFn(CGRectNull,
        kCGWindowListOptionIncludingWindow, (CGWindowID)windowID,
        kCGWindowImageBoundsIgnoreFraming);
    if (!image) return -1;

    CGImageRef finalImage = image;
    CGImageRef scaledImage = NULL;

    if (scale > 0.0f && scale < 1.0f) {
        scaledImage = scale_image(image, scale);
        if (scaledImage) {
            finalImage = scaledImage;
        }
    }

    result->width = (int)CGImageGetWidth(finalImage);
    result->height = (int)CGImageGetHeight(finalImage);

    int rc = encode_image(finalImage, format, quality, &result->data, &result->length);

    if (scaledImage) CGImageRelease(scaledImage);
    CGImageRelease(image);
    return rc;
}

int cg_capture_screen(int format, int quality, float scale,
                      ScreenshotResult* result) {
    CGWindowListCreateImageFunc captureFn = get_capture_func();
    if (!captureFn) return -1;

    CGImageRef image = captureFn(CGRectInfinite,
        kCGWindowListOptionOnScreenOnly, kCGNullWindowID,
        kCGWindowImageDefault);
    if (!image) return -1;

    CGImageRef finalImage = image;
    CGImageRef scaledImage = NULL;

    if (scale > 0.0f && scale < 1.0f) {
        scaledImage = scale_image(image, scale);
        if (scaledImage) {
            finalImage = scaledImage;
        }
    }

    result->width = (int)CGImageGetWidth(finalImage);
    result->height = (int)CGImageGetHeight(finalImage);

    int rc = encode_image(finalImage, format, quality, &result->data, &result->length);

    if (scaledImage) CGImageRelease(scaledImage);
    CGImageRelease(image);
    return rc;
}

float cg_get_menubar_height(void) {
    NSScreen *screen = [NSScreen mainScreen];
    if (!screen) return 25.0f; // fallback
    NSRect frame = [screen frame];
    NSRect visible = [screen visibleFrame];
    // Menu bar height = total height - visible height - visible origin offset
    // In macOS coordinates, origin is bottom-left. The menu bar is at the top.
    float menuBarHeight = (float)(frame.size.height - (visible.origin.y - frame.origin.y + visible.size.height));
    if (menuBarHeight < 20.0f) menuBarHeight = 25.0f; // sanity fallback
    return menuBarHeight;
}

float cg_get_display_width(void) {
    CGRect bounds = CGDisplayBounds(CGMainDisplayID());
    return (float)bounds.size.width;
}

int cg_capture_rect(float x, float y, float w, float h,
                    int format, int quality, float scale,
                    ScreenshotResult* result) {
    CGWindowListCreateImageFunc captureFn = get_capture_func();
    if (!captureFn) return -1;

    CGRect rect = CGRectMake(x, y, w, h);
    CGImageRef image = captureFn(rect,
        kCGWindowListOptionOnScreenOnly, kCGNullWindowID,
        kCGWindowImageDefault);
    if (!image) return -1;

    CGImageRef finalImage = image;
    CGImageRef scaledImage = NULL;

    if (scale > 0.0f && scale < 1.0f) {
        scaledImage = scale_image(image, scale);
        if (scaledImage) {
            finalImage = scaledImage;
        }
    }

    result->width = (int)CGImageGetWidth(finalImage);
    result->height = (int)CGImageGetHeight(finalImage);

    int rc = encode_image(finalImage, format, quality, &result->data, &result->length);

    if (scaledImage) CGImageRelease(scaledImage);
    CGImageRelease(image);
    return rc;
}

void cg_free_screenshot(ScreenshotResult* result) {
    if (result && result->data) {
        free(result->data);
        result->data = NULL;
    }
}

int cg_check_screen_recording(void) {
    return CGPreflightScreenCaptureAccess() ? 1 : 0;
}

int cg_request_screen_recording(void) {
    return CGRequestScreenCaptureAccess() ? 1 : 0;
}
