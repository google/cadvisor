export GO_FLAGS="-tags=libpfm,netgo -race"
export PACKAGES="sudo libpfm4"
export BUILD_PACKAGES="libpfm4 libpfm4-dev"
export CADVISOR_ARGS="-perf_events_config=perf/testing/perf-non-hardware.json"