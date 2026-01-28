package ui

import (
	"path/filepath"
	"fmt"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	"go-gnome-editor/internal/logic"
)

type ProjectTreeWidget struct {
	DockWidget *widgets.QDockWidget
	TreeView   *widgets.QTreeView
	Model      *gui.QStandardItemModel
	Editor     *EditorWindow 
}

func NewProjectTreeWidget(editor *EditorWindow) *ProjectTreeWidget {
	ptw := &ProjectTreeWidget{
		Editor: editor,
	}
	
	ptw.DockWidget = widgets.NewQDockWidget("Project", editor.Window, 0)
	ptw.DockWidget.SetAllowedAreas(core.Qt__LeftDockWidgetArea | core.Qt__RightDockWidgetArea)
	ptw.DockWidget.SetMinimumWidth(200)
	
	ptw.TreeView = widgets.NewQTreeView(nil)
	ptw.TreeView.SetHeaderHidden(true)
	ptw.TreeView.SetAnimated(true)
	ptw.TreeView.SetIndentation(15)

	ptw.TreeView.SetContextMenuPolicy(core.Qt__CustomContextMenu)
	ptw.TreeView.ConnectCustomContextMenuRequested(ptw.onContextMenu)
	
	ptw.Model = gui.NewQStandardItemModel(nil)
	ptw.TreeView.SetModel(ptw.Model)
	
	ptw.DockWidget.SetWidget(ptw.TreeView)
	ptw.TreeView.ConnectDoubleClicked(ptw.onItemDoubleClicked)
	
	return ptw
}

func (ptw *ProjectTreeWidget) Refresh() {
	ptw.Model.Clear()
	if !ptw.Editor.ProjectManager.IsActive { return }
	
	tree, err := ptw.Editor.ProjectManager.GetProjectTree()
	if err != nil || tree == nil { return }
	
	ptw.DockWidget.SetWindowTitle("Project: " + tree.Name)
	rootItem := ptw.Model.InvisibleRootItem()
	ptw.buildTreeItems(rootItem, tree.Children)
	ptw.TreeView.ExpandToDepth(0)
}

func (ptw *ProjectTreeWidget) buildTreeItems(parent *gui.QStandardItem, items []*logic.ProjectItem) {
	for _, item := range items {
		treeItem := gui.NewQStandardItem2(item.Name)
		treeItem.SetEditable(false)
		
		treeItem.SetData(core.NewQVariant1(item.Path), int(core.Qt__UserRole))
		
		// Metadata to distinguish dir/file safely
		// treeItem.SetData(core.NewQVariant11(item.IsDir), int(core.Qt__UserRole)+1)
		
		if item.IsDir {
			treeItem.SetIcon(ptw.getIcon("folder"))
		} else {
    		treeItem.SetIcon(ptw.getIconForFile(item.Name))
    	}
    
        if ptw.Editor.ProjectManager.IsFileInContext(item.Path) {
            treeItem.SetForeground(gui.NewQBrush3(gui.NewQColor2(core.Qt__darkGreen), core.Qt__SolidPattern))
        }

		if len(item.Children) > 0 {
			ptw.buildTreeItems(treeItem, item.Children)
		}
		
		parent.AppendRow2(treeItem)
	}
}

func (ptw *ProjectTreeWidget) getIcon(name string) *gui.QIcon {
	style := widgets.QApplication_Style()
	switch name {
	case "folder": return style.StandardIcon(widgets.QStyle__SP_DirIcon, nil, nil)
	default: return style.StandardIcon(widgets.QStyle__SP_FileIcon, nil, nil)
	}
}

func (ptw *ProjectTreeWidget) getIconForFile(filename string) *gui.QIcon {
	ext := filepath.Ext(filename)
	switch ext {
	case ".go": return ptw.getIcon("go")
	default: return ptw.getIcon("file")
	}
}

func (ptw *ProjectTreeWidget) onItemDoubleClicked(index *core.QModelIndex) {
	item := ptw.Model.ItemFromIndex(index)
	if item == nil { return }
	
	pathVariant := item.Data(int(core.Qt__UserRole))
	filePath := pathVariant.ToString()
	if filePath == "" { return }
	
	// Read metadata set in buildTreeItems
	isDir := item.Data(int(core.Qt__UserRole)+1).ToBool()

	if isDir {
		if ptw.TreeView.IsExpanded(index) {
			ptw.TreeView.Collapse(index)
		} else {
			ptw.TreeView.Expand(index)
		}
	} else {
		// Open in Tab - No need to check unsaved changes of other files
        ptw.Editor.TabManager.OpenFile(filePath)
    }
}

func (ptw *ProjectTreeWidget) Show() { ptw.DockWidget.Show() }
func (ptw *ProjectTreeWidget) Hide() { ptw.DockWidget.Hide() }
func (ptw *ProjectTreeWidget) IsVisible() bool { return ptw.DockWidget.IsVisible() }

func (ptw *ProjectTreeWidget) onContextMenu(pos *core.QPoint) {
	index := ptw.TreeView.IndexAt(pos)
	if !index.IsValid() { return }

	item := ptw.Model.ItemFromIndex(index)
	pathVariant := item.Data(int(core.Qt__UserRole))
	filePath := pathVariant.ToString()

	if filePath == "" { return }

	menu := widgets.NewQMenu(ptw.TreeView)

	// Action 1: Toggle LLM
	inContext := ptw.Editor.ProjectManager.IsFileInContext(filePath)
	ctxTitle := "Add to LLM Context"
	if inContext { ctxTitle = "Remove from LLM Context" }
	
	menu.AddAction(ctxTitle).ConnectTriggered(func(bool) {
		added := ptw.Editor.ProjectManager.ToggleContextFile(filePath)
		if added {
			item.SetForeground(gui.NewQBrush3(gui.NewQColor2(core.Qt__darkGreen), core.Qt__SolidPattern))
			ptw.Editor.Window.StatusBar().ShowMessage("Added: "+filepath.Base(filePath), 2000)
		} else {
			item.SetForeground(gui.NewQBrush3(gui.NewQColor2(core.Qt__black), core.Qt__SolidPattern))
			ptw.Editor.Window.StatusBar().ShowMessage("Removed context", 2000)
		}
	})

	menu.AddSeparator()

	// Action 2: Rename
	menu.AddAction("Rename").ConnectTriggered(func(bool) {
		oldName := filepath.Base(filePath)
		inputDialog := widgets.NewQInputDialog(ptw.Editor.Window, core.Qt__Dialog)
		inputDialog.SetWindowTitle("Rename")
		inputDialog.SetLabelText("New name:")
		inputDialog.SetTextValue(oldName)
		inputDialog.SetInputMode(widgets.QInputDialog__TextInput)

		ok := inputDialog.Exec() == int(widgets.QDialog__Accepted)
		newName := inputDialog.TextValue()

		if ok && newName != "" && newName != oldName {
			dir := filepath.Dir(filePath)
			newPath := filepath.Join(dir, newName)
			
			err := ptw.Editor.FileManager.RenameFile(filePath, newPath)
			if err != nil {
				widgets.QMessageBox_Critical(ptw.Editor.Window, "Error", err.Error(), widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
			} else {
				// Update Tab if open
				ptw.Editor.TabManager.UpdateFileAfterRename(filePath, newPath)
				
				// Update LLM Context
				if ptw.Editor.ProjectManager.IsFileInContext(filePath) {
					ptw.Editor.ProjectManager.ToggleContextFile(filePath) 
					ptw.Editor.ProjectManager.ToggleContextFile(newPath)
				}
				ptw.Refresh()
			}
		}
	})

	// Action 3: Delete
	menu.AddAction("Delete").ConnectTriggered(func(bool) {
		reply := widgets.QMessageBox_Question(
			ptw.Editor.Window, "Delete",
			fmt.Sprintf("Delete '%s'?", filepath.Base(filePath)),
			widgets.QMessageBox__Yes|widgets.QMessageBox__No,
			widgets.QMessageBox__No,
		)
		
		if reply == widgets.QMessageBox__Yes {
			err := ptw.Editor.FileManager.DeletePath(filePath)
			if err != nil {
				widgets.QMessageBox_Critical(ptw.Editor.Window, "Error", err.Error(), widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
			} else {
				if ptw.Editor.ProjectManager.IsFileInContext(filePath) {
					ptw.Editor.ProjectManager.ToggleContextFile(filePath)
				}
				// Optionally close tab if deleted file was open
				// ptw.Editor.TabManager.CloseFile(filePath) 
				ptw.Refresh()
			}
		}
	})

	menu.Exec2(gui.QCursor_Pos(), nil)
	// menu.Exec2(ptw.TreeView.MapToGlobal(pos), nil)
}
