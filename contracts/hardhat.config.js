require("@nomicfoundation/hardhat-toolbox");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.20",
  networks: {
    axon: {
      url: "http://localhost:8545",
      chainId: 9001,
      accounts: {
        mnemonic: "test test test test test test test test test test test junk",
      },
    },
  },
  paths: {
    sources: "./",
    tests: "./test-hardhat",
    cache: "./cache",
    artifacts: "./artifacts",
  },
};
