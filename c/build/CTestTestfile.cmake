# CMake generated Testfile for 
# Source directory: D:/vanni/idmix/c
# Build directory: D:/vanni/idmix/c/build
# 
# This file includes the relevant testing commands required for 
# testing this directory and lists subdirectories to be tested as well.
if(CTEST_CONFIGURATION_TYPE MATCHES "^([Dd][Ee][Bb][Uu][Gg])$")
  add_test(idmix_c_test "D:/vanni/idmix/c/build/Debug/idmix_c_test.exe")
  set_tests_properties(idmix_c_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/c/CMakeLists.txt;16;add_test;D:/vanni/idmix/c/CMakeLists.txt;0;")
elseif(CTEST_CONFIGURATION_TYPE MATCHES "^([Rr][Ee][Ll][Ee][Aa][Ss][Ee])$")
  add_test(idmix_c_test "D:/vanni/idmix/c/build/Release/idmix_c_test.exe")
  set_tests_properties(idmix_c_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/c/CMakeLists.txt;16;add_test;D:/vanni/idmix/c/CMakeLists.txt;0;")
elseif(CTEST_CONFIGURATION_TYPE MATCHES "^([Mm][Ii][Nn][Ss][Ii][Zz][Ee][Rr][Ee][Ll])$")
  add_test(idmix_c_test "D:/vanni/idmix/c/build/MinSizeRel/idmix_c_test.exe")
  set_tests_properties(idmix_c_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/c/CMakeLists.txt;16;add_test;D:/vanni/idmix/c/CMakeLists.txt;0;")
elseif(CTEST_CONFIGURATION_TYPE MATCHES "^([Rr][Ee][Ll][Ww][Ii][Tt][Hh][Dd][Ee][Bb][Ii][Nn][Ff][Oo])$")
  add_test(idmix_c_test "D:/vanni/idmix/c/build/RelWithDebInfo/idmix_c_test.exe")
  set_tests_properties(idmix_c_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/c/CMakeLists.txt;16;add_test;D:/vanni/idmix/c/CMakeLists.txt;0;")
else()
  add_test(idmix_c_test NOT_AVAILABLE)
endif()
