# Hardware Specifications

## CPU

- **Model**: Intel Core i7-10700K
- **Base Frequency**: 3.80GHz
- **Architecture**: x86_64
- **Cores**: 8 physical cores
- **Threads**: 16 (2 threads per core)
- **Scaling**: 87% (likely power-saving mode)

## GPU

- **Primary GPU**: AMD Radeon RX 5700 XT
- **Driver**: radeonsi (Mesa)
- **OpenGL Renderer**: AMD Radeon RX 5700 XT (radeonsi, navi10, LLVM 21.1.6, DRM 3.64)
- **OpenGL Version**: 4.6 (Compatibility Profile) Mesa 25.3.1-arch1.3
- **Vulkan Support**: Yes (vulkan-radeon installed)
- **OpenCL Support**: Yes (opencl-mesa installed)

### GPU Packages Installed

- mesa 1:25.3.1-3
- vulkan-radeon 1:25.3.1-3
- vulkan-mesa-implicit-layers 1:25.3.1-3
- opencl-mesa 1:25.3.1-3
- xf86-video-amdgpu 25.0.0-1.1
- linux-firmware-amdgpu 1:20251125-2

## Audio Hardware

### Audio Devices (from lspci)

- **Intel Audio**: Intel Corporation Comet Lake PCH cAVS (00:1f.3)
- **AMD HDMI Audio**: Advanced Micro Devices, Inc. [AMD/ATI] Navi 10 HDMI Audio (03:00.1)

### Audio Kernel Modules

Active sound modules:
- snd_sof_pci_intel_cnl (Intel Sound Open Firmware)
- snd_sof_intel_hda_generic
- snd_sof_intel_hda_common
- snd_sof (Sound Open Firmware core)
- snd_hda_codec_alc882 (Realtek ALC882 codec)
- snd_hda_codec_realtek_lib
- snd_hda_codec_generic
- snd_hda_codec_hdmi
- snd_hda_intel
- snd_usb_audio (USB audio support)
- soundwire_intel (Intel SoundWire)
- soundwire_cadence

## Memory (RAM)

- **Total Capacity**: 32 GB (4x 8GB modules)
- **Type**: DDR4 UDIMM (Unbuffered DIMM)
- **Speed**: 3600 MT/s (3600 MHz)
- **Configuration**: 4 slots, all populated
- **Manufacturer**: G Skill Intl
- **DRAM Manufacturer**: SK Hynix
- **Module Layout**:
  - ChannelA-DIMM1: 8 GB @ 3600 MT/s
  - ChannelA-DIMM2: 8 GB @ 3600 MT/s
  - ChannelB-DIMM1: 8 GB @ 3600 MT/s
  - ChannelB-DIMM2: 8 GB @ 3600 MT/s
- **Current Usage**: ~14 GB used, ~18 GB available (out of 32 GB total)
- **Swap**: 31 GB zram (0 GB used)

## Storage Devices

- **Primary SSD**: NVMe 931.5GB (Samsung or similar)
- **Additional HDDs**: 
  - 3.6TB (sda)
  - 3.6TB (sdb)
  - 931.5GB (sdc)
  - 1.8TB (sdd)

## Network

(To be filled in if needed)

## Peripherals

- **Input**: Standard keyboard/mouse (likely USB)
- **Camera**: UVC video device detected (uvcvideo module)


