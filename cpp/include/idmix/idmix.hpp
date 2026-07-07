#pragma once

#include "codec.hpp"
#include "idx.hpp"

#include <memory>
#include <random>
#include <string>
#include <vector>

namespace idmix {

class IdMix {
public:
    IdMix();
    explicit IdMix(const std::string& alphabet);
    IdMix(std::unique_ptr<Idx> idx, std::unique_ptr<ICodec> codec);

    static IdMix newDefault() { return IdMix(); }

    const Idx& idx() const { return *idx_; }
    const ICodec& codec() const { return *codec_; }

    std::string encode(const std::vector<Value>& values);
    std::string encodeWithVariant(int variantId, const std::vector<Value>& values);
    std::vector<Value> decode(const std::string& s);

private:
    std::unique_ptr<Idx> idx_;
    std::unique_ptr<ICodec> codec_;
    std::vector<uint8_t> encodeBinary(const std::vector<Value>& values, int variantId) const;
};

}  // namespace idmix
