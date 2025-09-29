import { withStyles } from "@mui/styles";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";
import SendIcon from "@mui/icons-material/Send";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import useMediaQuery from "@mui/material/useMediaQuery";
import { useTheme } from "@mui/material/styles";
import * as React from "react";
import SvgIcon from "@mui/material/SvgIcon";
import TreeView from "@mui/lab/TreeView";
import TreeItem, { treeItemClasses } from "@mui/lab/TreeItem";
import { ListItemIcon } from "@mui/material";
import { updatePool } from "../../memory";
import { forceUpdateApp } from "../../App";

function MinusSquare(props) {
  return (
    <SvgIcon fontSize="inherit" style={{ width: 14, height: 14 }} {...props}>
      {/* tslint:disable-next-line: max-line-length */}
      <path d="M22.047 22.074v0 0-20.147 0h-20.12v0 20.147 0h20.12zM22.047 24h-20.12q-.803 0-1.365-.562t-.562-1.365v-20.147q0-.776.562-1.351t1.365-.575h20.147q.776 0 1.351.575t.575 1.351v20.147q0 .803-.575 1.365t-1.378.562v0zM17.873 11.023h-11.826q-.375 0-.669.281t-.294.682v0q0 .401.294 .682t.669.281h11.826q.375 0 .669-.281t.294-.682v0q0-.401-.294-.682t-.669-.281z" />
    </SvgIcon>
  );
}

function PlusSquare(props) {
  return (
    <SvgIcon fontSize="inherit" style={{ width: 14, height: 14 }} {...props}>
      {/* tslint:disable-next-line: max-line-length */}
      <path d="M22.047 22.074v0 0-20.147 0h-20.12v0 20.147 0h20.12zM22.047 24h-20.12q-.803 0-1.365-.562t-.562-1.365v-20.147q0-.776.562-1.351t1.365-.575h20.147q.776 0 1.351.575t.575 1.351v20.147q0 .803-.575 1.365t-1.378.562v0zM17.873 12.977h-4.923v4.896q0 .401-.281.682t-.682.281v0q-.375 0-.669-.281t-.294-.682v-4.896h-4.923q-.401 0-.682-.294t-.281-.669v0q0-.401.281-.682t.682-.281h4.923v-4.896q0-.401.294-.682t.669-.281v0q.401 0 .682.281t.281.682v4.896h4.923q.401 0 .682.281t.281.682v0q0 .375-.281.669t-.682.294z" />
    </SvgIcon>
  );
}

function CloseSquare(props) {
  return (
    <SvgIcon
      className="close"
      fontSize="inherit"
      style={{ width: 14, height: 14 }}
      {...props}
    >
      {/* tslint:disable-next-line: max-line-length */}
      <path d="M17.485 17.512q-.281.281-.682.281t-.696-.268l-4.12-4.147-4.12 4.147q-.294.268-.696.268t-.682-.281-.281-.682.294-.669l4.12-4.147-4.12-4.147q-.294-.268-.294-.669t.281-.682.682-.281.696 .268l4.12 4.147 4.12-4.147q.294-.268.696-.268t.682.281 .281.669-.294.682l-4.12 4.147 4.12 4.147q.294.268 .294.669t-.281.682zM22.047 22.074v0 0-20.147 0h-20.12v0 20.147 0h20.12zM22.047 24h-20.12q-.803 0-1.365-.562t-.562-1.365v-20.147q0-.776.562-1.351t1.365-.575h20.147q.776 0 1.351.575t.575 1.351v20.147q0 .803-.575 1.365t-1.378.562v0z" />
    </SvgIcon>
  );
}

const StyledTreeItem = withStyles((theme) => ({
  iconContainer: {
    "& .close": {
      opacity: 0.3,
    },
  },
  group: {
    marginLeft: 7,
    paddingLeft: 18,
  },
}))((props) => <TreeItem {...props} />);

let idCounter = 0;
let anchorEl = null;
let setAnchorEl = null;
let createOpen = null;
let setCreateOpen = null;
let deleteOpen = null;
let setDeleteOpen = null;
let rightClickDisabled = false;
let isCut = false;

const handleClick = (event, path, fileSetter) => {
  if (rightClickDisabled) return;
  event.preventDefault();
  rightClickDisabled = true;
  setTimeout(() => {
    rightClickDisabled = false;
  }, 100);
  selectedFilePath = path;
  if (setAnchorEl !== null) setAnchorEl(event.currentTarget);
};

const handleClose = () => {
  if (setAnchorEl !== null) setAnchorEl(null);
};

const onOptionMenuItemSelected = (index, obj) => {
  handleClose();
  if (index === 0) {
    createType = "folder";
    setCreateOpen(true);
  } else if (index === 1) {
    createType = "file";
    setCreateOpen(true);
  } else if (index === 2) {
    pathToPaste = selectedFilePath;
    if (pathToPaste.includes("/")) {
      setPasteDisabled(false);
    } else {
      setPasteDisabled(true);
    }
    isCut = false;
  } else if (index === 3) {
    pathToPaste = selectedFilePath;
    if (pathToPaste.includes("/")) {
      setPasteDisabled(false);
    } else {
      setPasteDisabled(true);
    }
    isCut = true;
  } else if (index === 4) {
    let dst =
      selectedFilePath + pathToPaste.substring(pathToPaste.lastIndexOf("/"));
    const options = {
      method: "post",
      headers: {
        Accept: "application/json, text/plain, */*",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        src: pathToPaste,
        dst: dst,
      }),
    };
    fetch(isCut ? "../cut-file" : "../copy-file", options)
      .then((rawRes) => {
        selectedFileType = rawRes.headers.get("file-type");
        return rawRes.blob();
      })
      .then((blb) => {
        if (selectedFileType === "code") {
          selectedFilePath = obj.path;
          const reader = new FileReader();
          reader.addEventListener("loadend", (e) => {
            const text = e.srcElement.result;
            //codeSetter(text);
          });
          reader.readAsText(blb);
        }
      });
    if (isCut) {
      pathToPaste = dst;
      isCut = false;
    }
  } else if (index === 5) {
    setDeleteOpen(true);
  } else if (index === 6) {
    setRenameOpen(true);
  }
};

const handleCreate = () => {
  setCreateOpen(false);
  const options = {
    method: "post",
    headers: {
      Accept: "application/json, text/plain, */*",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      path: selectedFilePath + "/" + createFileNameValue,
    }),
  };
  fetch(createType == "file" ? "../make-file" : "../make-dir", options)
    .then((rawRes) => {
      return rawRes.json();
    })
    .then((res) => {});
};

const handleDelete = () => {
  setDeleteOpen(false);
  if (pathToPaste === selectedFilePath) {
    setPasteDisabled(true);
    pathToPaste = null;
  }
  const options = {
    method: "post",
    headers: {
      Accept: "application/json, text/plain, */*",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      path: selectedFilePath,
    }),
  };
  fetch("../delete-dir", options)
    .then((rawRes) => {
      return rawRes.json();
    })
    .then((res) => {});
};

let createFileNameValue = "";

const saveFieldValue = (text) => {
  createFileNameValue = text;
};

export function handleCurrentFilePathReset() {
  selectedFilePath = "";
}

function createView(obj) {
  return obj.children !== null && obj.children !== undefined ? (
    <StyledTreeItem
      style={{ color: "white" }}
      nodeId={idCounter}
      label={obj.name}
      onClick={() => {
        selectedFilePath = obj.path;
      }}
      onContextMenu={(event) => handleClick(event, obj.path)}
    >
      {obj.children.map((child) => {
        idCounter++;
        return createView(child);
      })}
    </StyledTreeItem>
  ) : (
    <StyledTreeItem
      style={{ color: "white" }}
      nodeId={idCounter}
      label={obj.name}
      onContextMenu={(event) => handleClick(event, obj.path)}
      onClick={(event) => {
        event.preventDefault();
        updatePool(obj.path, {path: obj.path});
        forceUpdateApp();
      }}
    ></StyledTreeItem>
  );
}
const saveRenameFieldValue = (value) => {
  renameFieldValue = value;
};

const handleRename = () => {
  setRenameOpen(false);
  if (selectedFilePath.endsWith("/"))
    selectedFilePath = selectedFilePath.substring(
      0,
      selectedFilePath.length - 1
    );
  let dst = "";
  if (selectedFilePath.includes("/"))
    dst = selectedFilePath.substring(0, selectedFilePath.lastIndexOf("/"));
  dst += "/" + renameFieldValue;
  const options = {
    method: "post",
    headers: {
      Accept: "application/json, text/plain, */*",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      src: selectedFilePath,
      dst: dst,
    }),
  };
  fetch("../rename-file", options);
};

let renameFieldValue = "";
let createType = "";
let selectedFilePath = "";
let selectedFileType = null;
let pathToPaste = null;
let pasteDisabled = true;
let setPasteDisabled = null;
let expanded = null;
let setExpanded = null;
let setRenameOpen = null;
let renameOpen = false;

export default function FileView(props) {
  [anchorEl, setAnchorEl] = React.useState(null);
  [pasteDisabled, setPasteDisabled] = React.useState(true);
  [createOpen, setCreateOpen] = React.useState(false);
  [deleteOpen, setDeleteOpen] = React.useState(false);
  [renameOpen, setRenameOpen] = React.useState(false);
  [expanded, setExpanded] = React.useState([0]);
  const theme = useTheme();
  const fullScreen = useMediaQuery(theme.breakpoints.down("sm"));
  idCounter = 0;
  return (
    <TreeView
      style={{
        height: "100%",
        flexGrow: 1,
        marginLeft: 16,
        marginTop: 16,
        overflow: "auto",
      }}
      expanded={expanded}
      onNodeToggle={(event, nodeIds) => {
        setExpanded(nodeIds);
      }}
      defaultCollapseIcon={<MinusSquare />}
      defaultExpandIcon={<PlusSquare />}
      defaultEndIcon={<CloseSquare />}
    >
      {createView(props.fileTree)}
      <Menu
        id="treeViewMenu"
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        <MenuItem onClick={() => onOptionMenuItemSelected(0, props.fileTree)}>
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">New Folder</Typography>
        </MenuItem>
        <MenuItem onClick={() => onOptionMenuItemSelected(1, props.fileTree)}>
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">New File</Typography>
        </MenuItem>
        <MenuItem onClick={() => onOptionMenuItemSelected(2, props.fileTree)}>
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">Copy</Typography>
        </MenuItem>
        <MenuItem onClick={() => onOptionMenuItemSelected(3, props.fileTree)}>
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">Cut</Typography>
        </MenuItem>
        <MenuItem
          disabled={pasteDisabled}
          onClick={() => onOptionMenuItemSelected(4, props.fileTree)}
        >
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">Paste</Typography>
        </MenuItem>
        <MenuItem onClick={() => onOptionMenuItemSelected(5, props.fileTree)}>
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">Delete</Typography>
        </MenuItem>
        <MenuItem onClick={() => onOptionMenuItemSelected(6, props.fileTree)}>
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <Typography variant="inherit">Rename</Typography>
        </MenuItem>
      </Menu>
      <Dialog
        open={renameOpen}
        onClose={() => setRenameOpen(false)}
        aria-labelledby="form-dialog-title"
      >
        <DialogTitle id="form-dialog-title">Rename file/folder</DialogTitle>
        <DialogContent>
          <DialogContentText>Enter file/folder name:</DialogContentText>
          <TextField
            onChange={(event) => saveRenameFieldValue(event.target.value)}
            autoFocus
            margin="dense"
            id="name"
            label="Name"
            type="text"
            fullWidth
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRenameOpen(false)} color="primary">
            Cancel
          </Button>
          <Button onClick={() => handleRename()} color="primary">
            Rename
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        aria-labelledby="form-dialog-title"
      >
        <DialogTitle id="form-dialog-title">Create file/folder</DialogTitle>
        <DialogContent>
          <DialogContentText>Enter file/folder name:</DialogContentText>
          <TextField
            onChange={(event) => saveFieldValue(event.target.value)}
            autoFocus
            margin="dense"
            id="name"
            label="Name"
            type="text"
            fullWidth
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)} color="primary">
            Cancel
          </Button>
          <Button onClick={() => handleCreate()} color="primary">
            Create
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        fullScreen={fullScreen}
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        aria-labelledby="responsive-dialog-title"
      >
        <DialogTitle id="responsive-dialog-title">
          {"Delete file/folder"}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            Do you really want to delete this file/folder ?
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button
            autoFocus
            onClick={() => setDeleteOpen(false)}
            color="primary"
          >
            Cancel
          </Button>
          <Button onClick={() => handleDelete()} color="primary" autoFocus>
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </TreeView>
  );
}

export function openCreateFile() {
  createType = "file";
  setCreateOpen(true);
}

export function handleSwitchProject() {
  setExpanded([]);
}
