# CMake generated Testfile for 
# Source directory: D:/vanni/idmix/cpp
# Build directory: D:/vanni/idmix/cpp/build
# 
# This file includes the relevant testing commands required for 
# testing this directory and lists subdirectories to be tested as well.
if(CTEST_CONFIGURATION_TYPE MATCHES "^([Dd][Ee][Bb][Uu][Gg])$")
  add_test(idmix_test "D:/vanni/idmix/cpp/build/Debug/idmix_test.exe")
  set_tests_properties(idmix_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/cpp/CMakeLists.txt;19;add_test;D:/vanni/idmix/cpp/CMakeLists.txt;0;")
elseif(CTEST_CONFIGURATION_TYPE MATCHES "^([Rr][Ee][Ll][Ee][Aa][Ss][Ee])$")
  add_test(idmix_test "D:/vanni/idmix/cpp/build/Release/idmix_test.exe")
  set_tests_properties(idmix_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/cpp/CMakeLists.txt;19;add_test;D:/vanni/idmix/cpp/CMakeLists.txt;0;")
elseif(CTEST_CONFIGURATION_TYPE MATCHES "^([Mm][Ii][Nn][Ss][Ii][Zz][Ee][Rr][Ee][Ll])$")
  add_test(idmix_test "D:/vanni/idmix/cpp/build/MinSizeRel/idmix_test.exe")
  set_tests_properties(idmix_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/cpp/CMakeLists.txt;19;add_test;D:/vanni/idmix/cpp/CMakeLists.txt;0;")
elseif(CTEST_CONFIGURATION_TYPE MATCHES "^([Rr][Ee][Ll][Ww][Ii][Tt][Hh][Dd][Ee][Bb][Ii][Nn][Ff][Oo])$")
  add_test(idmix_test "D:/vanni/idmix/cpp/build/RelWithDebInfo/idmix_test.exe")
  set_tests_properties(idmix_test PROPERTIES  _BACKTRACE_TRIPLES "D:/vanni/idmix/cpp/CMakeLists.txt;19;add_test;D:/vanni/idmix/cpp/CMakeLists.txt;0;")
else()
  add_test(idmix_test NOT_AVAILABLE)
endif()
