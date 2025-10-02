import * as React from "react";
import Box from "@mui/material/Box";
import Drawer from "@mui/material/Drawer";
import Button from "@mui/material/Button";
import List from "@mui/material/List";
import Divider from "@mui/material/Divider";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import { Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, FormControlLabel, Radio, RadioGroup, TextField } from "@mui/material";
import { colors } from "../../App";

import AddIcon from '@mui/icons-material/Add';
import FormatListBulletedIcon from '@mui/icons-material/FormatListBulleted';
import SettingsIcon from '@mui/icons-material/Settings';
import Header from '../../images/header.jpg';

let createFileNameValue = "";

export default function MainDrawer(props) {
  const [createOpen, setCreateOpen] = React.useState(false);
  const [projectTypeValue, setProjectTypeValue] = React.useState("Desktop");
  const [projectsOpen, setProjectsOpen] = React.useState(false);
  const [projectsList, setProjectsList] = React.useState(null);

  const handleClose = () => {
    props.setDrawerOpen(false);
  };
  const saveFieldValue = (text) => {
    createFileNameValue = text;
  };
  const handleCreate = () => {
    const options = {
      method: "post",
      headers: {
        Accept: "application/json, text/plain, */*",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        name: createFileNameValue,
        projectType: projectTypeValue,
      }),
    };
    fetch("../create-project", options).then((res) => {
      setCreateOpen(false);
    });
  };
  const onOptionMenuItemSelected = (index) => {
    if (index === 0) {
      handleClose();
      setCreateOpen(true);
    } else if (index === 1) {
      handleClose();
      setProjectsOpen(true);
      const options = {
        method: "post",
        headers: {
          Accept: "application/json, text/plain, */*",
          "Content-Type": "application/json",
        },
      };
      fetch("../get-projects", options)
        .then((rawRes) => rawRes.json())
        .then((res) => {
          setProjectsList(res);
        });
    }
  };
  const handleChange = event => {
    setProjectTypeValue(event.target.value);
};

let projectCounter = 0;

  const list = () => {
    let anchor = "left";
    return (
      <Box
        sx={{ width: anchor === "top" || anchor === "bottom" ? "auto" : 350 }}
        role="presentation"
      >
        <img style={{width: '100%', height: 240}} src={Header} alt={'header'} />
        <List>
          <ListItem key={"New Project"} disablePadding onClick={() => onOptionMenuItemSelected(0)}>
            <ListItemButton>
              <ListItemIcon><AddIcon style={{fill: '#fff'}}/></ListItemIcon>
              <ListItemText primary={"New Project"} style={{color: '#fff'}}/>
            </ListItemButton>
          </ListItem>
          <ListItem key={"Projects List"} disablePadding onClick={() => onOptionMenuItemSelected(1)}>
            <ListItemButton>
              <ListItemIcon><FormatListBulletedIcon style={{fill: '#fff'}}/></ListItemIcon>
              <ListItemText primary={"Projects List"} style={{color: '#fff'}}/>
            </ListItemButton>
          </ListItem>
          <ListItem key={"Exit"} disablePadding onClick={() => onOptionMenuItemSelected(2)}>
            <ListItemButton>
              <ListItemIcon><SettingsIcon style={{fill: '#fff'}}/></ListItemIcon>
              <ListItemText primary={"Exit"} style={{color: '#fff'}}/>
            </ListItemButton>
          </ListItem>
        </List>
      </Box>
    );
  };

  return (
    <div>
      <React.Fragment key={"left"}>
        <Drawer
          anchor={"left"}
          open={props.drawerOpen}
          onClose={() => props.setDrawerOpen(false)}
          PaperProps={{
            style: {
              backgroundColor: colors.colorDark3
            }
          }}
        >
          {list("left")}
        </Drawer>
      </React.Fragment>
      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        aria-labelledby="form-dialog-title"
      >
        <DialogTitle id="form-dialog-title">Create new project</DialogTitle>
        <DialogContent>
          <DialogContentText>Enter project name:</DialogContentText>
          <TextField
            onChange={(event) => saveFieldValue(event.target.value)}
            autoFocus
            margin="dense"
            id="name"
            label="Name"
            type="text"
            fullWidth
          />
          <RadioGroup
            style={{ marginTop: 16 }}
            aria-label="Project Type"
            name="ProjectType"
            value={projectTypeValue}
            onChange={handleChange}
          >
            <FormControlLabel
              value="Desktop"
              control={<Radio />}
              label="Desktop"
            />
            <FormControlLabel value="Web" control={<Radio />} label="Web" />
            <FormControlLabel
              value="Mobile"
              control={<Radio />}
              label="Mobile"
            />
          </RadioGroup>
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
        maxWidth="xs"
        aria-labelledby="confirmation-dialog-title"
        onClose={() => setProjectsOpen(false)}
        open={projectsOpen}
      >
        <DialogTitle id="confirmation-dialog-title">Projects</DialogTitle>
        <DialogContent dividers>
          <List component="div" role="list">
            {projectsList === null
              ? null
              : projectsList.map((projectName) => {
                  const projectId = projectCounter;
                  projectCounter++;
                  return (
                    <ListItem
                      button
                      divider
                      aria-haspopup="true"
                      aria-label="project"
                      onClick={() => {
                        const options = {
                          method: "post",
                          headers: {
                            Accept: "application/json, text/plain, */*",
                            "Content-Type": "application/json",
                          },
                          body: JSON.stringify({
                            projectName: projectsList[projectId],
                          }),
                        };
                        fetch("../switch-project", options);
                      }}
                      role="listitem"
                    >
                      <ListItemText
                        primary={projectName}
                        secondary={projectName}
                      />
                    </ListItem>
                  );
                })}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setProjectsOpen(false)} color="primary">
            Ok
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}
