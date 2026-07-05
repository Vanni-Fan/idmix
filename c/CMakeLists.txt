cmake_minimum_required(VERSION 3.16)
project(idmix_c VERSION 0.2.0 LANGUAGES C)

add_library(idmix_c STATIC src/idmix.c)
add_library(idmix::c ALIAS idmix_c)

target_include_directories(idmix_c PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)
target_compile_features(idmix_c PUBLIC c_std_99)

enable_testing()
add_executable(idmix_c_test tests/test_idmix.c)
target_link_libraries(idmix_c_test PRIVATE idmix_c)
add_test(NAME idmix_c_test COMMAND idmix_c_test)

install(TARGETS idmix_c EXPORT idmix_cTargets
    ARCHIVE DESTINATION lib
    LIBRARY DESTINATION lib
    RUNTIME DESTINATION bin
)
install(FILES include/idmix.h DESTINATION include)
install(EXPORT idmix_cTargets
    FILE idmix_cConfig.cmake
    NAMESPACE idmix::
    DESTINATION lib/cmake/idmix_c
)
