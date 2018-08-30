set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${PROJECT_SOURCE_DIR}/../cmake/")
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_VERBOSE_MAKEFILE ON)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR})
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR})

set(SGX_SDK /opt/intel/sgxsdk)
set(SGX_ARCH x64)
set(SGX_MODE HW) # SGX mode: sim, hw
set(SGX_BUILD PRERELEASE)
set(SGX_SSL /opt/intel/sgxssl)