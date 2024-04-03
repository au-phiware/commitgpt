{ pkgs ? import <nixpkgs> {} }:

pkgs.buildGoModule rec {
  pname = "commitgpt";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-1wycFQdf6sudxnH10xNz1bppRDCQjCz33n+ugP74SdQ=";

  nativeBuildInputs = with pkgs; [
    makeWrapper
  ] ++ buildInputs;

  buildInputs = with pkgs; [
    pandoc
    git
  ];

  postInstall = ''
    wrapProgram $out/bin/commitgpt \
      --set PATH ${pkgs.lib.makeBinPath buildInputs} \
      --run 'export ANTHROPIC_API_KEY="$(${pkgs.libsecret}/bin/secret-tool lookup anthropic-api-key commitgpt)"'
  '';

  meta = with pkgs.lib; {
    description = "A tool to generate commit messages using Claude";
    license = licenses.mit;
    maintainers = with maintainers; [ "Corin Lawson" ];
    platforms = platforms.unix;
  };
}
