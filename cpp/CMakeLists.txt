cmake_minimum_required(VERSION 3.16)
project(idmix VERSION 0.2.0 LANGUAGES CXX)

add_library(idmix STATIC
    src/idmix.cpp
    src/xid_codec.cpp
)
add_library(idmix::idmix ALIAS idmix)

target_include_directories(idmix PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)
target_compile_features(idmix PUBLIC cxx_std_17)

enable_testing()
add_executable(idmix_test tests/test_idmix.cpp)
target_link_libraries(idmix_test PRIVATE idmix)
add_test(NAME idmix_test COMMAND idmix_test)

install(TARGETS idmix EXPORT idmixTargets
    ARCHIVE DESTINATION lib
    LIBRARY DESTINATION lib
    RUNTIME DESTINATION bin
)
install(DIRECTORY include/ DESTINATION include)
install(EXPORT idmixTargets
    FILE idmixConfig.cmake
    NAMESPACE idmix::
    DESTINATION lib/cmake/idmix
)
