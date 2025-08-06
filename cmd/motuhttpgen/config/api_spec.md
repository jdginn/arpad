## Global Settings

### uid

Type: string
Permission: r
Available since global version: 0.0.
Description: The UID of the device. The UID is a 16 digit hexadecimal string that uniquely identifies this device on AVB
networks.

### ext/caps/avb

Type: semver_opt
Permission: r
Available since global version: 0.0.
Description: The version of the avb section. If this path is absent, the device does not have the paths in the avb section.

### ext/caps/router

Type: semver_opt
Permission: r
Available since global version: 0.0.
Description: The version of the router section. If this path is absent, the device does not have the paths in the router
section.

### ext/caps/mixer

Type: semver_opt
Permission: r
Available since global version: 0.0.
Description: The version of the mixer section. If this path is absent, the device does not have the paths in the mixer
section.

## AVB (Audio Video Bridging) Settings

The avb section of the datastore is special because it includes information on all AVB devices in the target device's AVB network,
in addition to the local parameters of that device. The list of all devices exists at avb/devs. Each device in that list maintains a
separate subtree, containing all AVB parameters, located at avb/<uid>. Any AVB-capable device -- even those not created by
MOTU -- will appear in the avb section, although MOTU-only parameters such as apiversion and url will only appear for MOTU
devices.

### avb/devs

Type: string_list
Permission: r
Available since avb version: 0.0.
Description: A list of UIDs for AVB devices on the same network as this device.

### avb/<uid>/entity_model_id_h

Type: int
Permission: r
Available since avb version: 0.0.
Description: The vendor id of the connected AVB device.

### avb/<uid>/entity_model_id_l

Type: int
Permission: r
Available since avb version: 0.0.
Description: The model id of the connected AVB device.

### avb/<uid>/entity_name

Type: string
Permission: rw
Available since avb version: 0.0.
Description: The human readable name of the connected AVB device. On MOTU devices, this may be changed by the
user or an API client (e.g., "My 1248").

### avb/<uid>/model_name

Type: string
Permission: r
Available since avb version: 0.0.
Description: The human readable model name of the connected AVB device (e.g., "1248").

### avb/<uid>/hostname

Type: string_opt
Permission: r
Available since avb version: 0.0.
Description: The sanitized hostname assigned to this device. This is only valid for MOTU devices. This may be different
from entity_name in that it won't have spaces or non-ascii characters (e.g., "My-1248").

### avb/<uid>/master_clock/capable

Type: int_bool
Permission: r
Available since avb version: 0.0.
Description: True if this device supports MOTU Master Clock. MOTU Master Clock is a set of special datastore keys in the
avb section that allows one device to quickly become the clock source of many others.

### avb/<uid>/master_clock/uid

Type: string_opt
Permission: rw
Available since avb version: 0.0.
Description: The UID of the device the master_clock stream is connected to, or the empty string if there is no connection.
Only available for devices that are Master Clock capable (see master_clock/capable above).

### avb/<uid>/vendor_name

Type: string
Permission: r
Available since avb version: 0.0.
Description: The human readable vendor name of the connected AVB device (e.g., "MOTU").

### avb/<uid>/firmware_version

Type: string
Permission: r
Available since avb version: 0.0.
Description: The human readable firmware version number of the connected AVB device. For MOTU devices, this will be
a semver.

### avb/<uid>/serial_number

Type: string
Permission: r
Available since avb version: 0.0.
Description: The human readable serial number of the connected AVB device.

### avb/<uid>/controller_ignore

Type: int_bool
Permission: r
Available since avb version: 0.0.
Description: True if this device should be ignored. If true, clients should not show this device in their UI.

### avb/<uid>/acquired_id

Type: string
Permission: r
Available since avb version: 0.0.
Description: The controller UID of the controller that acquired this box, or the empty string if no controller has acquired it.
Acquisition is a part of the AVB standard that allows a controller to prevent other controllers from making changes on this
device. You cannot initiate an acquisition from the datastore API, but you should avoid making changes on a device that
has been acquired elsewhere.

### avb/<uid>/apiversion

Type: semver_opt
Permission: r
Available since avb version: 0.0.
Description: The global datastore API version of the device. This path is only valid for MOTU devices.

### avb/<uid>/url

Type: string_opt
Permission: r
Available since avb version: 0.0.
Description: The canonical url of the device. This path is only valid for MOTU devices.

### avb/<uid>/current_configuration

Type: int
Permission: rw
Available since avb version: 0.0.
Description: The index of the currently active device configuration. MOTU devices only have one configuration, index 0.
Other devices may have multiple available configurations.

### avb/<uid>/cfg/<index>/object_name

Type: string
Permission: r
Available since avb version: 0.0.
Description: The name of the configuration with the given index.

### avb/<uid>/cfg/<index>/identify

Type: int_bool
Permission: rw
Available since avb version: 0.0.
Description: True if the configuration is in identify mode. What identify mode means depends on the device. For MOTU
devices, identify will flash the front panel backlight.

### avb/<uid>/cfg/<index>/current_sampling_rate

Type: int
Permission: rw
Available since avb version: 0.0.
Description: The sampling rate of the configuration with the given index.

### avb/<uid>/cfg/<index>/sample_rates

Type: int_list
Permission: r
Available since avb version: 0.0.
Description: A list of allowed sample rates for the configuration with the given index.

### avb/<uid>/cfg/<index>/clock_source_index

Type: int
Permission: rw
Available since avb version: 0.0.
Description: The currently chosen clock source for the configuration with the given index.

### avb/<uid>/cfg/<index>/clock_sources/num

Type: int
Permission: r
Available since avb version: 0.0.
Description: The number of available clock sources for the given configuration.

### avb/<uid>/cfg/<index>/clock_sources/<index>/object_name

Type: string
Permission: r
Available since avb version: 0.0.
Description: The name of the clock source with the given index.

### avb/<uid>/cfg/<index>/clock_sources/<index>/type

Type: string
Permission: r
Available since avb version: 0.0.
Description: The type of the clock source with the given index. The value will be one of "internal", "external", or "stream".

### avb/<uid>/cfg/<index>/clock_sources/<index>/stream_id

Type: int_opt
Permission: r
Available since avb version: 0.0.
Description: If the type of the clock source is "stream", the id of the stream from which it derives its clock. This path is only
valid if the clock is a stream.

### avb/<uid>/cfg/<index>/input_streams/<index>/talker

Type: string_pair

Permission: rw
Available since avb version: 0.0.
Description: The talker for the given input stream. The first element of the pair is the device UID, the second element of
the pair is the stream ID that this stream is connected to.

### ext/clockLocked

Type: int_bool
Permission: r
Available since avb version: 0.0.
Description: True if the clock is locked.

## Routing and I/O Settings

### ext/wordClockMode

Type: string
Permission: rw
Available since router version: 0.2.
Description: "1x" if the word clock out should always be a 1x rate or "follow" if it should always follow the system clock

### ext/wordClockThru

Type: string
Permission: rw
Available since router version: 0.2.
Description: "thru" if the word clock output should be the same as the word clock input or "out" if it should be determined
by the system clock

### ext/smuxPerBank

Type: int_bool
Permission: r
Available since router version: 0.2.
Description: True if each optical bank has its own SMUX setting

### ext/vlimit/lookahead

Type: int_bool_opt
Permission: rw
Available since router version: 0.0.
Description: True if vLimit lookahead is enabled. vLimit lookahead provides better input limiting, at the cost of small
amounts of extra latency. This path is only present on devices with access to vLimit.

### ext/enableHostVolControls

Type: int_bool
Permission: rw
Available since router version: 0.1.
Description: True if the comptuter is allowed to control the volumes of comptuer-to-device streams.

### ext/maxUSBToHost

Type: int
Permission: rw
Available since router version: 0.1.
Description: Valid only when this device is connected to the computer via USB. This chooses the max number of

channels/max sample rate tradeoff for the to/from computer input/output banks.

### ext/<ibank_or_obank>/<index>/name

Type: string
Permission: r
Available since router version: 0.0.
Description: The name of the input or output bank

### ext/<ibank_or_obank>/<index>/maxCh

Type: int
Permission: r
Available since router version: 0.0.
Description: The maximum possible number of channels in the input or output bank.

### ext/<ibank_or_obank>/<index>/numCh

Type: int
Permission: r
Available since router version: 0.0.
Description: The number of channels available in this bank at its current sample rate.

### ext/<ibank_or_obank>/<index>/userCh

Type: int
Permission: rw
Available since router version: 0.0.
Description: The number of channels that the user has enabled for this bank.

### ext/<ibank_or_obank>/<index>/calcCh

Type: int
Permission: r
Available since router version: 0.0.
Description: The number of channels that are actually active. This is always the minimum of
ext/<ibank_or_obank>/<index>/userCh and ext/<ibank_or_obank>/<index>/userCh.

### ext/<ibank_or_obank>/<index>/smux

Type: string
Permission: rw
Available since router version: 0.2.
Description: For Optical banks, either "toslink" or "adat"

### ext/ibank/<index>/madiClock

Type: string
Permission: r
Available since router version: 0.2.
Description: For MADI input banks, this is the 2x clock mode of the input stream-- "1x" for 48/44.1kHz frame clock, or "2x"
for 88.2/96kHz frame clock

### ext/obank/<index>/madiClock

Type: string
Permission: rw
Available since router version: 0.2.

Description: For MADI output banks, this is the 2x clock mode of the output stream-- "1x" for 48/44.1kHz frame clock, or
"2x" for 88.2/96kHz frame clock

### ext/ibank/<index>/madiFormat

Type: int
Permission: r
Available since router version: 0.2.
Description: 56 or 64 representing 56 or 64 MADI channels at 1x, 28 or 32 channels at 2x, or 14 or 16 channels at 4x,
respectively

### ext/obank/<index>/madiFormat

Type: int
Permission: rw
Available since router version: 0.2.
Description: 56 or 64 representing 56 or 64 MADI channels at 1x, 28 or 32 channels at 2x, or 14 or 16 channels at 4x,
respectively

### ext/<ibank_or_obank>/<index>/ch/<index>/name

Type: string
Permission: rw
Available since router version: 0.0.
Description: The channel's name.

### ext/obank/<index>/ch/<index>/src

Type: int_pair_opt
Permission: rw
Available since router version: 0.0.
Description: If the output channel is connected to an input bank, a ":" separated pair in the form " :
", otherwise, if unrouted, an empty string.

### ext/<ibank_or_obank>/<index>/ch/<index>/phase

Type: int_bool_opt
Permission: rw
Available since router version: 0.0.
Description: True if the signal has its phase inverted. This is only applicable to some input or output channels.

### ext/<ibank_or_obank>/<index>/ch/<index>/pad

Type: int_bool_opt
Permission: rw
Available since router version: 0.0.
Description: True if the 20 dB pad is engaged. This is only applicable to some input or output channels.

### ext/ibank/<index>/ch/<index>/48V

Type: int_bool_opt
Permission: rw
Available since router version: 0.0.
Description: True if the 48V phantom power is engaged. This is only applicable to some input channels.

### ext/ibank/<index>/ch/<index>/vlLimit

Type: int_bool_opt
Permission: rw
Available since router version: 0.0.
Description: True if the vLimit limiter is engaged. This is only applicable to some input channels.

### ext/ibank/<index>/ch/<index>/vlClip

Type: int_bool_opt
Permission: rw
Available since router version: 0.0.
Description: True if vLimit clip is engaged. This is only applicable to some input channels.

### ext/<ibank_or_obank>/<index>/ch/<index>/trim

Type: int_opt
Permission: rw
Available since router version: 0.0.
Description: A dB-value for how much to trim this input or output channel. The range of this parameter is indicated by
ext/<ibank_or_obank>/<index>/ch/<index>/trimRange. Only available for certain input or output channels.

### ext/<ibank_or_obank>/<index>/ch/<index>/trimRange

Type: int_pair_opt
Permission: rw
Available since router version: 0.0.
Description: A pair of the minimum followed by maximum values allowed for the trim parameter on the input or output
channel.

### ext/<ibank_or_obank>/<index>/ch/<index>/stereoTrim

Type: int_opt
Permission: rw
Available since router version: 0.0.
Description: A dB-value for how much to trim this input or output channel. This stereo trim affect both this channel and the
next one. The range of this parameter is indicated by ext/<ibank_or_obank>/<index>/ch/<index>/stereoTrimRange. Only
available for certain input or output channels.

### ext/<ibank_or_obank>/<index>/ch/<index>/stereoTrimRange

Type: int_pair_opt
Permission: rw
Available since router version: 0.0.
Description: A pair of the minimum followed by maximum values allowed for the stereoTrim parameter on the input or
output channel.

### ext/<ibank_or_obank>/<index>/ch/<index>/connection

Type: int_bool_opt
Permission: r
Available since router version: 0.0.
Description: True if the channel has a physical connector plugged in (e.g., an audio jack). This information may not be
available for all banks or devices.

## Mixer Settings

The mixer section as described is only valid for the current mixer version, 1.0. In future versions, paths, types, or valid parameter
ranges may change.

### mix/ctrls/dsp/usage

Type: int
Permission: r
Available since mixer version: 1.0.
Description: The approximate percentage of DSP resources used for mixing and effects.

### mix/ctrls/<effect_resource>/avail

Type: int_bool_opt
Permission: r
Available since mixer version: 1.0.
Description: True if there are enough DSP resources to enable one more of the given effect.

### mix/chan/<index>/matrix/aux/<index>/send

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: linear

### mix/chan/<index>/matrix/group/<index>/send

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: linear

### mix/chan/<index>/matrix/reverb/<index>/send

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: linear

### mix/chan/<index>/matrix/aux/<index>/pan

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: pan

### mix/chan/<index>/matrix/group/<index>/pan

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:

Unit: pan

### mix/chan/<index>/matrix/reverb/<index>/pan

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: pan

### mix/chan/<index>/hpf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/hpf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/chan/<index>/eq/highshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/eq/highshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/chan/<index>/eq/highshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/chan/<index>/eq/highshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.
Maximum Value:

Unit: octaves

### mix/chan/<index>/eq/highshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.
Possible Values: Shelf=0,Para=

### mix/chan/<index>/eq/mid1/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/eq/mid1/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/chan/<index>/eq/mid1/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/chan/<index>/eq/mid1/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.
Maximum Value:
Unit: octaves

### mix/chan/<index>/eq/mid2/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/eq/mid2/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/chan/<index>/eq/mid2/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/chan/<index>/eq/mid2/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.
Maximum Value:
Unit: octaves

### mix/chan/<index>/eq/lowshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/eq/lowshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/chan/<index>/eq/lowshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/chan/<index>/eq/lowshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.
Maximum Value:
Unit: octaves

### mix/chan/<index>/eq/lowshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.
Possible Values: Shelf=0,Para=

### mix/chan/<index>/gate/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/gate/release

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: ms

### mix/chan/<index>/gate/threshold

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: linear

### mix/chan/<index>/gate/attack

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: ms

### mix/chan/<index>/comp/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/comp/release

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: ms

### mix/chan/<index>/comp/threshold

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/chan/<index>/comp/ratio

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:

### mix/chan/<index>/comp/attack

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: ms

### mix/chan/<index>/comp/trim

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/chan/<index>/comp/peak

Type: real_enum
Permission: rw
Available since mixer version: 1.0.
Possible Values: RMS=0,Peak=

### mix/chan/<index>/matrix/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/matrix/solo

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/matrix/mute

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/chan/<index>/matrix/pan

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -

Maximum Value:
Unit: pan

### mix/chan/<index>/matrix/fader

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: linear

### mix/main/<index>/eq/highshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/main/<index>/eq/highshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/main/<index>/eq/highshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/main/<index>/eq/highshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.
Maximum Value:
Unit: octaves

### mix/main/<index>/eq/highshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.
Possible Values: Shelf=0,Para=

### mix/main/<index>/eq/mid1/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/main/<index>/eq/mid1/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/main/<index>/eq/mid1/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/main/<index>/eq/mid1/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.
Maximum Value:
Unit: octaves

### mix/main/<index>/eq/mid2/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.

### mix/main/<index>/eq/mid2/freq

Type: int
Permission: rw
Available since mixer version: 1.0.
Minimum Value:
Maximum Value:
Unit: Hz

### mix/main/<index>/eq/mid2/gain

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: -
Maximum Value:
Unit: dB

### mix/main/<index>/eq/mid2/bw

Type: real
Permission: rw
Available since mixer version: 1.0.
Minimum Value: 0.

Maximum Value: 3
Unit: octaves

### mix/main/<index>/eq/lowshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/main/<index>/eq/lowshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/main/<index>/eq/lowshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/main/<index>/eq/lowshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/main/<index>/eq/lowshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/main/<index>/leveler/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/main/<index>/leveler/makeup

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/main/<index>/leveler/reduction

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/main/<index>/leveler/limit

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/main/<index>/matrix/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/main/<index>/matrix/mute

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/main/<index>/matrix/fader

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/aux/<index>/eq/highshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/eq/highshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/aux/<index>/eq/highshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20

Maximum Value: 20
Unit: dB

### mix/aux/<index>/eq/highshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/aux/<index>/eq/highshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/aux/<index>/eq/mid1/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/eq/mid1/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/aux/<index>/eq/mid1/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/aux/<index>/eq/mid1/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/aux/<index>/eq/mid2/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/eq/mid2/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/aux/<index>/eq/mid2/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/aux/<index>/eq/mid2/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/aux/<index>/eq/lowshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/eq/lowshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/aux/<index>/eq/lowshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/aux/<index>/eq/lowshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01

Maximum Value: 3
Unit: octaves

### mix/aux/<index>/eq/lowshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/aux/<index>/matrix/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/matrix/prefader

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/matrix/panner

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/matrix/mute

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/aux/<index>/matrix/fader

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/group/<index>/matrix/aux/<index>/send

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/group/<index>/matrix/reverb/<index>/send

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0

Maximum Value: 4
Unit: linear

### mix/group/<index>/eq/highshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/eq/highshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/group/<index>/eq/highshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/group/<index>/eq/highshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/group/<index>/eq/highshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/group/<index>/eq/mid1/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/eq/mid1/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/group/<index>/eq/mid1/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/group/<index>/eq/mid1/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/group/<index>/eq/mid2/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/eq/mid2/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/group/<index>/eq/mid2/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/group/<index>/eq/mid2/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/group/<index>/eq/lowshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/eq/lowshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/group/<index>/eq/lowshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/group/<index>/eq/lowshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/group/<index>/eq/lowshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/group/<index>/leveler/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/leveler/makeup

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/group/<index>/leveler/reduction

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/group/<index>/leveler/limit

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/matrix/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/matrix/solo

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/matrix/prefader

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/matrix/panner

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/matrix/mute

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/group/<index>/matrix/fader

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/reverb/<index>/matrix/aux/<index>/send

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/reverb/<index>/matrix/reverb/<index>/send

Type: real

Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/reverb/<index>/eq/highshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/eq/highshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/reverb/<index>/eq/highshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/reverb/<index>/eq/highshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/reverb/<index>/eq/highshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/reverb/<index>/eq/mid1/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/eq/mid1/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0

Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/reverb/<index>/eq/mid1/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/reverb/<index>/eq/mid1/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/reverb/<index>/eq/mid2/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/eq/mid2/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/reverb/<index>/eq/mid2/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/reverb/<index>/eq/mid2/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/reverb/<index>/eq/lowshelf/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/eq/lowshelf/freq

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 20
Maximum Value: 20000
Unit: Hz

### mix/reverb/<index>/eq/lowshelf/gain

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -20
Maximum Value: 20
Unit: dB

### mix/reverb/<index>/eq/lowshelf/bw

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0.01
Maximum Value: 3
Unit: octaves

### mix/reverb/<index>/eq/lowshelf/mode

Type: real_enum
Permission: rw
Available since mixer version: 1.0.0
Possible Values: Shelf=0,Para=1

### mix/reverb/<index>/leveler/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/leveler/makeup

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/reverb/<index>/leveler/reduction

Type: real
Permission: rw

Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/reverb/<index>/leveler/limit

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/matrix/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/matrix/solo

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/matrix/prefader

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/matrix/panner

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/matrix/mute

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/matrix/fader

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/reverb/<index>/reverb/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/reverb/<index>/reverb/reverbtime

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 100
Maximum Value: 60000
Unit: ms

### mix/reverb/<index>/reverb/hf

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 500
Maximum Value: 15000
Unit: Hz

### mix/reverb/<index>/reverb/mf

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 500
Maximum Value: 15000
Unit: Hz

### mix/reverb/<index>/reverb/predelay

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 500
Unit: ms

### mix/reverb/<index>/reverb/mfratio

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 1
Maximum Value: 100
Unit: %

### mix/reverb/<index>/reverb/hfratio

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 1
Maximum Value: 100
Unit: %

### mix/reverb/<index>/reverb/tailspread

Type: int
Permission: rw
Available since mixer version: 1.0.0

Minimum Value: -100
Maximum Value: 100
Unit: %

### mix/reverb/<index>/reverb/mod

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 100
Unit: %

### mix/monitor/<index>/matrix/enable

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/monitor/<index>/matrix/mute

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0

### mix/monitor/<index>/matrix/fader

Type: real
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: 0
Maximum Value: 4
Unit: linear

### mix/monitor/<index>/assign

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -2
Maximum Value: 4096

### mix/monitor/<index>/override

Type: int
Permission: rw
Available since mixer version: 1.0.0
Minimum Value: -1
Maximum Value: 4096

### mix/monitor/<index>/auto

Type: real_bool
Permission: rw
Available since mixer version: 1.0.0
