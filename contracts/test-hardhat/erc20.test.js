const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("TestERC20", function () {
  let token, owner, addr1;

  beforeEach(async function () {
    [owner, addr1] = await ethers.getSigners();
    const ERC20 = await ethers.getContractFactory("TestERC20");
    token = await ERC20.deploy("Test Token", "TT", 18, ethers.parseEther("1000"));
    await token.waitForDeployment();
  });

  it("should have correct name and symbol", async function () {
    expect(await token.name()).to.equal("Test Token");
    expect(await token.symbol()).to.equal("TT");
  });

  it("should assign total supply to deployer", async function () {
    const balance = await token.balanceOf(owner.address);
    expect(balance).to.equal(ethers.parseEther("1000"));
  });

  it("should transfer tokens", async function () {
    await token.transfer(addr1.address, ethers.parseEther("100"));
    expect(await token.balanceOf(addr1.address)).to.equal(ethers.parseEther("100"));
    expect(await token.balanceOf(owner.address)).to.equal(ethers.parseEther("900"));
  });

  it("should emit Transfer event", async function () {
    await expect(token.transfer(addr1.address, ethers.parseEther("50")))
      .to.emit(token, "Transfer")
      .withArgs(owner.address, addr1.address, ethers.parseEther("50"));
  });
});

describe("Precompile: IAgentRegistry", function () {
  const REGISTRY = "0x0000000000000000000000000000000000000801";

  it("should respond to isAgent query", async function () {
    const abi = ["function isAgent(address) view returns (bool)"];
    const registry = new ethers.Contract(REGISTRY, abi, ethers.provider);
    const result = await registry.isAgent("0x0000000000000000000000000000000000000001");
    expect(typeof result).to.equal("boolean");
  });
});

describe("Precompile: IAgentReputation", function () {
  const REPUTATION = "0x0000000000000000000000000000000000000802";

  it("should return reputation score", async function () {
    const abi = ["function getReputation(address) view returns (uint64)"];
    const rep = new ethers.Contract(REPUTATION, abi, ethers.provider);
    const score = await rep.getReputation("0x0000000000000000000000000000000000000001");
    expect(score).to.be.a("bigint");
  });
});
