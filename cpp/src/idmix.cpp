#include "idmix/idmix.hpp"

#include <stdexcept>

namespace idmix {

IdMix::IdMix() : idx_(std::make_unique<Idx>()), codec_(std::make_unique<RadixCodec>(DEFAULT_ALPHABET)) {}

IdMix::IdMix(const std::string& alphabet)
    : idx_(std::make_unique<Idx>()), codec_(std::make_unique<RadixCodec>(alphabet)) {}

IdMix::IdMix(std::unique_ptr<Idx> idx, std::unique_ptr<ICodec> codec)
    : idx_(std::move(idx)), codec_(std::move(codec)) {
    if (!idx_ || !codec_) throw std::invalid_argument("idx and codec cannot be null");
}

std::vector<uint8_t> IdMix::encodeBinary(const std::vector<Value>& values, int variantId) const {
    return idx_->encodeWithVariant(variantId, values);
}

std::string IdMix::encodeWithVariant(int variantId, const std::vector<Value>& values) {
    if (values.empty()) throw std::invalid_argument("at least one value is required");
    auto data = encodeBinary(values, variantId);
    return codec_->encode(data);
}

std::string IdMix::encode(const std::vector<Value>& values) {
    if (values.empty()) throw std::invalid_argument("at least one value is required");
    static thread_local std::mt19937 rng{std::random_device{}()};
    std::uniform_int_distribution<int> dist(0, idx_->maxVariants() - 1);
    return encodeWithVariant(dist(rng), values);
}

std::vector<Value> IdMix::decode(const std::string& s) {
    auto data = codec_->decode(s);
    return idx_->decode(data);
}

}  // namespace idmix
