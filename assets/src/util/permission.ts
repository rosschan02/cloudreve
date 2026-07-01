import { FileResponse } from "../api/explorer.ts";
import CrUri, { Filesystem } from "./uri.ts";

// canCopyMoveTo checks if the files can be copied or moved to the destination.
export function canCopyMoveTo(files: FileResponse[], dst: string, isCopy: boolean): boolean {
  const dstUri = new CrUri(dst);
  const srcUri = new CrUri(files[0].path);
  if (isCopy) {
    // Copy is allowed freely between "my" files and the group share area (either
    // direction), so users can grab a copy of a group template into their own space.
    // Mirrors the backend canMoveOrCopyTo.
    const inMyOrGroup = (fs: string) => fs == Filesystem.my || fs == Filesystem.group;
    return inMyOrGroup(srcUri.fs()) && inMyOrGroup(dstUri.fs());
  } else {
    switch (srcUri.fs()) {
      case Filesystem.my:
        return dstUri.fs() == Filesystem.my || dstUri.fs() == Filesystem.trash;
      case Filesystem.trash:
        return dstUri.fs() == Filesystem.my;
      case Filesystem.group:
        // Move only within the group share area (cross-fs move would change ownership).
        return dstUri.fs() == Filesystem.group;
    }
  }

  return false;
}
