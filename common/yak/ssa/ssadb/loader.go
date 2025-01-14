package ssadb

import (
	"context"

	"github.com/jinzhu/gorm"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils/bizhelper"
)

func YieldIrCodesProgramName(db *gorm.DB, ctx context.Context, program string) chan *IrCode {
	db = db.Model(&IrCode{}).Where("program_name = ?", program)
	return yieldIrCodes(db, ctx)
}

func yieldIrCodes(db *gorm.DB, ctx context.Context) chan *IrCode {
	db = db.Model(&IrCode{})
	outC := make(chan *IrCode)
	go func() {
		defer close(outC)

		var page = 1
		for {
			var items []*IrCode
			if _, b := bizhelper.Paging(db, page, 100, &items); b.Error != nil {
				log.Errorf("paging failed: %s", b.Error)
				return
			}

			page++
			for _, d := range items {
				select {
				case <-ctx.Done():
					return
				case outC <- d:
				}
			}

			if len(items) < 100 {
				return
			}
		}
	}()
	return outC
}

func yieldIrVariables(db *gorm.DB, ctx context.Context) chan int64 {
	db = db.Model(&IrVariable{})
	outC := make(chan int64)
	go func() {
		defer close(outC)

		filter := make(map[int64]struct{})

		var page = 1
		for {
			var items []*IrVariable
			if _, b := bizhelper.Paging(db, page, 100, &items); b.Error != nil {
				log.Errorf("paging failed: %s", b.Error)
				return
			}

			page++
			for _, d := range items {
				for _, id := range d.InstructionID {
					if _, ok := filter[id]; ok {
						continue
					}
					filter[id] = struct{}{}

					select {
					case <-ctx.Done():
						return
					case outC <- id:
					}
				}
			}

			if len(items) < 100 {
				return
			}
		}
	}()
	return outC
}

func ExactSearchVariable(DB *gorm.DB, isMatchMember bool, value string) chan int64 {
	db := DB.Model(&IrVariable{})
	if isMatchMember {
		db = db.Where("("+
			"slice_member_name = ? "+
			"OR field_member_name = ?"+
			")", value, value)
	} else {
		db = db.Where("( variable_name = ? )", value)
	}
	return yieldIrVariables(db, context.Background())
}

func GlobSearchVariable(DB *gorm.DB, isMatchMember bool, value string) chan int64 {
	db := DB.Model(&IrVariable{})
	if isMatchMember {
		db = db.Where("("+
			"slice_member_name GLOB ? "+
			"OR field_member_name GLOB ?"+
			")", value, value)
	} else {
		db = db.Where("( variable_name GLOB ? )", value)
	}
	return yieldIrVariables(db, context.Background())
}
func RegexpSearchVariable(DB *gorm.DB, isMatchMember bool, value string) chan int64 {
	db := DB.Model(&IrVariable{})
	if isMatchMember {
		db = db.Where("("+
			"slice_member_name REGEXP ? "+
			"OR field_member_name REGEXP ?"+
			")", value, value)
	} else {
		db = db.Where("( variable_name REGEXP ? )", value)
	}
	return yieldIrVariables(db, context.Background())
}
