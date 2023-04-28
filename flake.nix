{
  description =
    "An AWS code artifact proxy to allow unauthenticated read access to your code artifacts";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    {
      overlays.default = (final: prev: {
        inherit (self.packages.${final.system}) aws-codeartifact-proxy;
      });
    } // flake-utils.lib.eachDefaultSystem (system:
      let
        package-name = "aws-codeartifact-proxy";
        urn = "github.com/sktan/${package-name}";

        pkgs = import nixpkgs { inherit system; };

        aws-codeartifact-proxy = pkgs.buildGoModule {
          pname = package-name;
          name = package-name;
          src = ./src;
          vendorSha256 = "3MO+mRCstXw0FfySiyMSs1vaao7kUYIyJB2gAp1IE48=";
          meta = with pkgs.lib; {
            description =
              "An AWS code artifact proxy to allow unauthenticated read access to your code artifacts";
            homepage = "https://${urn}";
            license = licenses.mit;
            maintainers = with maintainers; [ lafrenierejm ];
          };
        };
      in rec {
        packages = flake-utils.lib.flattenTree {
          # `nix build .#aws-codeartifact-proxy`
          inherit aws-codeartifact-proxy;
          # `nix build`
          default = aws-codeartifact-proxy;
          # Build an OCI image.
          # `nix build .#aws-codeartifact-proxy-oci`
          aws-codeartifact-proxy-docker = pkgs.dockerTools.buildLayeredImage {
            name = "aws-codeartifact-proxy";
            tag = "latest";
            config = {
              Entrypoint =
                [ "${aws-codeartifact-proxy}/bin/aws-codeartifact-proxy" ];
              WorkingDir = "/data";
              Volumes = { "/data" = { }; };
            };
          };
        };

        # `nix run`
        apps.default =
          flake-utils.lib.mkApp { drv = packages.aws-codeartifact-proxy; };

        # `nix develop`
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go gopls gotools go-tools nixfmt ];
        };
      });
}
