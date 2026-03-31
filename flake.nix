{
  description = "Codex Online - Action MMO dev environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixpkgs-stable.url = "github:NixOS/nixpkgs/nixos-24.11";
  };

  outputs = { self, nixpkgs, nixpkgs-stable }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      pkgs-stable = nixpkgs-stable.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [
          # Client
          godot_4

          # Assets
          blender
          uv            # uvx for blender-mcp

          # Server
          go
          gopls         # Go LSP
          gotools       # goimports, etc.
          delve         # Go debugger

          # Data stores (local dev)
          redis
          postgresql

          # Tools
          jq            # used in hooks
          just          # task runner
        ];

        env = {
          GOPATH = "$PWD/.go";
        };

        shellHook = ''
          echo "Codex Online dev shell"
          echo "  godot : $(godot --version 2>/dev/null || echo 'available')"
          echo "  blender: $(blender --version 2>&1 | head -1) (from nixos-24.11)"
          echo "  go    : $(go version 2>/dev/null | cut -d' ' -f3)"
          echo "  redis : $(redis-server --version 2>/dev/null | cut -d' ' -f1-3)"
        '';
      };
    };
}
