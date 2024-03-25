package ssadb

import (
	"sync"

	"github.com/jinzhu/gorm"
)

type IrCode struct {
	gorm.Model

	ProgramName string `json:"program_name" gorm:"index"`
	PackageName string `json:"package_name" gorm:"index"`

	// source code
	SourceCodeFile      string `json:"source_code_file"`
	SourceCodeStartLine int64  `json:"source_code_start_line"`
	SourceCodeEndLine   int64  `json:"source_code_end_line"`
	SourceCodeStartCol  int64  `json:"source_code_start_col"`
	SourceCodeEndCol    int64  `json:"source_code_end_col"`
	SourceCodeHash      string `json:"source_code_hash"`

	// opcode
	Opcode         int64  `json:"opcode"`
	OpcodeName     string `json:"opcode_name"`
	OpcodeOperator string `json:"opcode_operator"`

	// basic info
	Name             string `json:"name"`
	VerboseName      string `json:"verbose_name"`
	ShortVerboseName string `json:"short_verbose_name"`

	// any IrCode in one function
	CurrentBlock    int64 `json:"current_block"`
	CurrentFunction int64 `json:"current_function"`

	// FunctionDefs
	IsFunction       bool       `json:"is_function"`
	FormalArgs       Int64Slice `json:"formal_args" gorm:"type:text"`
	IsVariadic       bool       `json:"is_variadic"`
	ReturnCodes      Int64Slice `json:"return_codes" gorm:"type:text"`
	SideEffects      Int64Slice `json:"side_effects" gorm:"type:text"`
	IsExternal       bool       `json:"is_external"`
	IsCalledBy       Int64Slice `json:"is_called_by" gorm:"type:text"`
	CodeBlocks       Int64Slice `json:"code_blocks" gorm:"type:text"`
	EnterBlock       int64      `json:"enter_block"`
	ExitBlock        int64      `json:"exit_block"`
	DeferBlock       int64      `json:"defer_block"`
	ChildrenFunction Int64Slice `json:"children_function" gorm:"type:text"`
	ParentFunction   int64      `json:"parent_function"`

	// block
	IsBlock   bool       `json:"is_block"`
	PredBlock Int64Slice `json:"pred_block" gorm:"type:text"`
	SuccBlock Int64Slice `json:"succ_block" gorm:"type:text"`
	Phis      Int64Slice `json:"phis_in_block" gorm:"type:text"`

	// Use-Def Chains Relation
	Defs  Int64Slice `json:"defs" gorm:"type:text"`
	Users Int64Slice `json:"users" gorm:"type:text"`

	// OOP Supporting
	IsObject       bool
	IsObjectMember bool
	ObjectMembers  Int64Map `json:"object_members" gorm:"type:text"`
	ObjectParent   int64    `json:"object_parent"`

	// Maskable
	MaskedCodes Int64Slice `json:"masked_codes" gorm:"type:text"`
	IsMasked    bool       `json:"is_masked"`

	// Called
	IsCalled   bool       `json:"is_called"`
	ActualArgs Int64Slice `json:"actual_args" gorm:"type:text"`
}

func emptyIrCode() *IrCode {
	return &IrCode{}
}

var verifyExisted = new(sync.Once)

func RequireIrCode(db *gorm.DB) (uint, *IrCode) {
	verifyExisted.Do(func() {
		db.AutoMigrate(&IrCode{})
	})
	db = db.Model(&IrCode{})
	// save new ircode
	ircode := emptyIrCode()
	db.Create(ircode)
	return ircode.ID, ircode
}

func GetIrCodeById(db *gorm.DB, id uint) *IrCode {
	return db.Model(&IrCode{}).Where("id = ?", id).First(&IrCode{}).Value.(*IrCode)
}

func GetIrByVariable(db *gorm.DB, program, name string) *IrCode {
	var r IrCode
	if err := db.Model(&IrCode{}).Where("program_name = ?", program).Where("variable like ?", "%"+name+"%").First(&r).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil
		}
	}
	return &r
}

func DeleteProgram(db *gorm.DB, program string) {
	db.Model(&IrCode{}).Where("program_name = ?", program).Unscoped().Delete(&IrCode{})
}